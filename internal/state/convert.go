package state

import (
	"fmt"

	external "github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/internal/scope"
)

type Converter struct {
	BlueprintInterpreter BlueprintInterpreter
}

type BlueprintInterpreter interface {
	InterpretBlueprint(source external.BlueprintSource, input map[string]any) (external.Blueprint, error)
}

func (c *Converter) ConvertStmt(s *State, sc *scope.Scope, stmt external.Stmt) (any, error) {
	switch stmt := stmt.Value.(type) {
	case external.DeclareBuild:
		return c.ConvertBuildStmt(s, sc, stmt)
	case external.DeclareResource:
		return c.ConvertResourceStmt(s, sc, stmt)
	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

func (c *Converter) ConvertBuildStmt(s *State, sc *scope.Scope, build external.DeclareBuild) (StmtBuild, error) {
	blueprint, err := c.BlueprintInterpreter.InterpretBlueprint(build.BlueprintSource, build.Input)
	if err != nil {
		return StmtBuild{}, err
	}

	runtimeInput, err := c.ConvertMapExpr(build.Name, build.Runtimeinput)
	if err != nil {
		return StmtBuild{}, fmt.Errorf("converting runtime input: %s", err)
	}

	exists, err := c.ConvertBoolExpr(build.Name, build.Exists)
	if err != nil {
		return StmtBuild{}, err
	}

	buildID := sc.ComponentID(build.Name)

	var stmts []any
	subG := sc.Sub(build.Name)
	for _, stmt := range blueprint.Stmts {
		s, err := c.ConvertStmt(s, subG, stmt)
		if err != nil {
			return StmtBuild{}, err
		}

		stmts = append(stmts, s)
	}

	b := StmtBuild{
		ID:           buildID,
		Name:         build.Name,
		Exists:       exists,
		BuildID:      sc.ID(),
		RuntimeInput: runtimeInput,
		Stmts:        stmts,
	}

	sc.SetBuild(buildID, b)
	s.Builds[buildID] = &BuildState{
		name: build.Name,
	}

	return b, nil
}

func (c *Converter) ConvertResourceStmt(s *State, sc *scope.Scope, stmt external.DeclareResource) (StmtResource, error) {
	resource, err := c.ConvertResourceExpr(stmt.Name, stmt.Resource)
	if err != nil {
		return StmtResource{}, err
	}

	exists, err := c.ConvertBoolExpr(stmt.Name, stmt.Exists)
	if err != nil {
		return StmtResource{}, err
	}

	resourceID := sc.ComponentID(stmt.Name)
	r := StmtResource{
		ID:       resourceID,
		Name:     stmt.Name,
		Exists:   exists,
		BuildID:  sc.ID(),
		Resource: resource,
	}

	sc.SetResource(resourceID, r)
	s.Resources[resourceID] = &ResourceState{
		name: stmt.Name,
	}
	return r, nil
}

func (c *Converter) ConvertProviderExpr(name string, expr external.Expr) (Expr[Provider], error) {
	switch value := expr.Value.(type) {
	case external.Provider:
		n, err := c.ConvertStringExpr(name, value.Name)
		if err != nil {
			return nil, err
		}

		v, err := c.ConvertStringExpr(name, value.Version)
		if err != nil {
			return nil, err
		}

		return ExprProvider{
			Name:    n,
			Version: v,
		}, nil
	default:
		return nil, fmt.Errorf("invalid provider expr: %T", expr)
	}
}

func (c *Converter) ConvertResourceExpr(name string, expr external.Expr) (Expr[Resource], error) {
	switch value := expr.Value.(type) {
	case external.Resource:
		identifier, err := c.ConvertAnyExpr(name, value.Identifier)
		if err != nil {
			return nil, err
		}

		config, err := c.ConvertAnyExpr(name, value.Config)
		if err != nil {
			return nil, err
		}

		t, err := c.ConvertStringExpr(name, value.Type)
		if err != nil {
			return nil, err
		}

		p, err := c.ConvertProviderExpr(name, value.Provider)
		if err != nil {
			return nil, err
		}

		return ExprResource{
			Name:       name,
			Provider:   p,
			Type:       t,
			Identifier: identifier,
			Config:     config,
		}, nil
	default:
		return nil, fmt.Errorf("invalid resource expr: %T", expr)
	}
}

func (c *Converter) ConvertAnyExpr(name string, expr external.Expr) (Expr[any], error) {
	switch expr.Value.(type) {
	case external.StringLiteral:
		expr, err := c.ConvertStringExpr(name, expr)
		if err != nil {
			return nil, err
		}

		return ExprAny[string]{Value: expr}, nil
	case external.BoolLiteral:
		expr, err := c.ConvertBoolExpr(name, expr)
		if err != nil {
			return nil, err
		}

		return ExprAny[bool]{Value: expr}, nil
	case external.MapCollection:
		expr, err := c.ConvertMapExpr(name, expr)
		if err != nil {
			return nil, err
		}

		return ExprAny[map[string]any]{Value: expr}, nil
	case external.Resource:
		expr, err := c.ConvertResourceExpr(name, expr)
		if err != nil {
			return nil, err
		}

		return ExprAny[Resource]{Value: expr}, nil
	default:
		return nil, fmt.Errorf("invalid expr: %T", expr.Value)
	}
}

func (c *Converter) ConvertMapExpr(name string, expr external.Expr) (ExprMap, error) {
	switch value := expr.Value.(type) {
	case external.MapCollection:
		m := ExprMap{}
		for k, v := range value.Value {
			val, err := c.ConvertAnyExpr(name, v)
			if err != nil {
				return nil, err
			}

			m[ExprLiteral[string]{Value: k}] = val
		}
		return m, nil
	default:
		return nil, fmt.Errorf("%s: invalid map expr: %T", name, expr)
	}
}

func (c *Converter) ConvertBoolExpr(name string, expr external.Expr) (Expr[bool], error) {
	switch value := expr.Value.(type) {
	case external.BoolLiteral:
		return ExprLiteral[bool]{Value: value.Value}, nil
	default:
		return nil, fmt.Errorf("invalid bool expr: %T", expr)
	}
}

func (c *Converter) ConvertStringExpr(name string, expr external.Expr) (Expr[string], error) {
	switch value := expr.Value.(type) {
	case external.StringLiteral:
		return ExprLiteral[string]{Value: value.Value}, nil
	default:
		return nil, fmt.Errorf("invalid string expr: %T", expr)
	}
}

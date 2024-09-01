package plan

import (
	"errors"
	"fmt"
	"log/slog"

	external "github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/internal/scope"
)

type Converter struct {
	BlueprintInterpreter BlueprintInterpreter
	Logger               *slog.Logger
}

type BlueprintInterpreter interface {
	InterpretBlueprint(source external.BlueprintSource, input map[string]any) (external.Blueprint, error)
}

func (c *Converter) ConvertStmt(p *Plan, sc *scope.Scope, parentID string, stmt external.Stmt) (any, error) {
	switch stmt := stmt.Value.(type) {
	case external.DeclareBuild:
		return c.ConvertBuildStmt(p, sc, parentID, stmt)
	case external.DeclareResource:
		return c.ConvertResourceStmt(p, sc, parentID, stmt)
	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

func (c *Converter) ConvertBuildStmt(p *Plan, sc *scope.Scope, parentID string, build external.DeclareBuild) (StmtBuild, error) {
	// TODO: better validation function.
	if build.Exists.IsEmpty() {
		return StmtBuild{}, errors.New("must provide exists")
	}

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

	// buildID := sc.ComponentID(build.Name)
	buildID := fmt.Sprintf("%s.%s", parentID, build.Name)

	var stmts []any
	for _, stmt := range blueprint.Stmts {
		s, err := c.ConvertStmt(p, sc, buildID, stmt)
		if err != nil {
			return StmtBuild{}, err
		}

		stmts = append(stmts, s)
	}

	b := StmtBuild{
		ID:           buildID,
		Name:         build.Name,
		Exists:       exists,
		BuildID:      parentID,
		RuntimeInput: runtimeInput,
		Stmts:        stmts,
	}

	sc.SetBuild(parentID, buildID, b)
	p.Builds[buildID] = NewBuildPlan(build.Name)

	return b, nil
}

func (c *Converter) ConvertResourceStmt(p *Plan, sc *scope.Scope, parentID string, stmt external.DeclareResource) (StmtResource, error) {
	// TODO: Validate

	exists, err := c.ConvertBoolExpr(stmt.Name, stmt.Exists)
	if err != nil {
		return StmtResource{}, err
	}

	t, err := c.ConvertStringExpr(stmt.Name, stmt.Type)
	if err != nil {
		return StmtResource{}, err
	}

	provider, err := c.ConvertProviderExpr(stmt.Name, stmt.Provider)
	if err != nil {
		return StmtResource{}, err
	}

	id, err := c.ConvertAnyExpr(stmt.Name, stmt.Identifier)
	if err != nil {
		return StmtResource{}, err
	}

	config, err := c.ConvertAnyExpr(stmt.Name, stmt.Config)
	if err != nil {
		return StmtResource{}, err
	}

	// resourceID := sc.ComponentID(stmt.Name)
	resourceID := fmt.Sprintf("%s.%s", parentID, stmt.Name)
	r := StmtResource{
		ID:      resourceID,
		Name:    stmt.Name,
		BuildID: parentID,

		Exists:     exists,
		Type:       t,
		Provider:   provider,
		Identifier: id,
		Config:     config,
	}

	sc.SetResource(parentID, resourceID, r)
	p.Resources[resourceID] = NewResourcePlan(stmt.Name)
	return r, nil
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

		return ExprAny[map[Maybe[string]]Maybe[any]]{Value: expr}, nil
	default:
		return nil, fmt.Errorf("invalid expr: %T", expr.Value)
	}
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

func (c *Converter) ConvertStringExpr(name string, expr external.Expr) (Expr[string], error) {
	switch value := expr.Value.(type) {
	case external.StringLiteral:
		return ExprLiteral[string]{Value: value.Value}, nil
	default:
		return nil, fmt.Errorf("invalid string expr: %T", expr)
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

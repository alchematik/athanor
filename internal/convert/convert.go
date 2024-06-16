package convert

import (
	"errors"
	"fmt"
	"log/slog"

	external "github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/scope"
	"github.com/alchematik/athanor/internal/state"
)

type Converter struct {
	Logger               *slog.Logger
	BlueprintInterpreter BlueprintInterpreter
}

type BlueprintInterpreter interface {
	InterpretBlueprint(source external.BlueprintSource, input map[string]any) (external.Blueprint, error)
}

func ConvertBoolExpr(name string, expr external.Expr) (ast.Expr[bool], error) {
	switch value := expr.Value.(type) {
	case external.BoolLiteral:
		return ast.Literal[bool]{Value: value.Value}, nil
	default:
		return nil, fmt.Errorf("invalid bool expr: %T", expr)
	}
}

func ConvertMapExpr(name string, expr external.Expr) (ast.Expr[map[string]any], error) {
	switch value := expr.Value.(type) {
	case external.MapCollection:
		m := ast.Map[any]{
			Value: map[ast.Expr[string]]ast.Expr[any]{},
		}
		for k, v := range value.Value {
			val, err := ConvertAnyExpr(name, v)
			if err != nil {
				return nil, err
			}

			m.Value[ast.Literal[string]{Value: k}] = val
		}
		return m, nil
	default:
		return nil, fmt.Errorf("%s: invalid map expr: %T", name, expr)
	}
}

func ConvertResourceExpr(name string, expr external.Expr) (ast.Expr[state.Resource], error) {
	switch value := expr.Value.(type) {
	case external.Resource:
		identifier, err := ConvertAnyExpr(name, value.Identifier)
		if err != nil {
			return nil, err
		}

		config, err := ConvertAnyExpr(name, value.Config)
		if err != nil {
			return nil, err
		}

		return ast.ResourceExpr{
			Name:       name,
			Identifier: identifier,
			Config:     config,
		}, nil
	case external.GetResource:
		from, err := ConvertAnyExpr(name, value.From)
		if err != nil {
			return nil, err
		}

		return ast.GetResource{Name: value.Name, From: from}, nil
	default:
		return nil, fmt.Errorf("invalid resource expr: %T", expr)
	}
}

func (c *Converter) ConvertStmt(s *state.State, sc *scope.Scope, stmt external.Stmt) (any, error) {
	switch stmt := stmt.Value.(type) {
	case external.DeclareBuild:
		return c.ConvertBuildStmt(s, sc, stmt)
	case external.DeclareResource:
		return c.ConvertResourceStmt(s, sc, stmt)
	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

func (c *Converter) ConvertResourceStmt(s *state.State, sc *scope.Scope, stmt external.DeclareResource) (ast.StmtResource, error) {
	if stmt.Resource.IsEmpty() {
		// TODO: Add validation error type.
		return ast.StmtResource{}, errors.New("must provide resource")
	}

	if stmt.Exists.IsEmpty() {
		return ast.StmtResource{}, errors.New("must provide exists")
	}

	resource, err := ConvertResourceExpr(stmt.Name, stmt.Resource)
	if err != nil {
		return ast.StmtResource{}, err
	}

	exists, err := ConvertBoolExpr(stmt.Name, stmt.Exists)
	if err != nil {
		return ast.StmtResource{}, err
	}

	resourceID := sc.ComponentID(stmt.Name)
	r := ast.StmtResource{
		ID:       resourceID,
		Name:     stmt.Name,
		Exists:   exists,
		BuildID:  sc.ID(),
		Resource: resource,
	}

	sc.SetResource(resourceID, r)
	s.Resources[resourceID] = &state.ResourceState{
		Resource: state.Resource{
			Name: stmt.Name,
		},
	}
	return r, nil
}

func (c *Converter) ConvertBuildStmt(s *state.State, sc *scope.Scope, build external.DeclareBuild) (ast.StmtBuild, error) {
	// TODO: better validation function.
	if build.Exists.IsEmpty() {
		return ast.StmtBuild{}, errors.New("must provide exists")
	}

	blueprint, err := c.BlueprintInterpreter.InterpretBlueprint(build.BlueprintSource, build.Input)
	if err != nil {
		return ast.StmtBuild{}, err
	}

	runtimeInput, err := ConvertMapExpr(build.Name, build.Runtimeinput)
	if err != nil {
		return ast.StmtBuild{}, fmt.Errorf("converting runtime input: %s", err)
	}

	exists, err := ConvertBoolExpr(build.Name, build.Exists)
	if err != nil {
		return ast.StmtBuild{}, err
	}

	buildID := sc.ComponentID(build.Name)

	var stmts []any
	subG := sc.Sub(build.Name)
	for _, stmt := range blueprint.Stmts {
		s, err := c.ConvertStmt(s, subG, stmt)
		if err != nil {
			return ast.StmtBuild{}, err
		}

		stmts = append(stmts, s)
	}

	b := ast.StmtBuild{
		ID:           buildID,
		Name:         build.Name,
		Exists:       exists,
		BuildID:      sc.ID(),
		RuntimeInput: runtimeInput,
		Stmts:        stmts,
	}

	sc.SetBuild(buildID, b)
	s.Builds[buildID] = &state.BuildState{
		Build: state.Build{
			Name: build.Name,
		},
	}

	return b, nil
}

func ConvertAnyExpr(name string, expr external.Expr) (ast.Expr[any], error) {
	switch expr.Value.(type) {
	case external.StringLiteral:
		expr, err := ConvertStringExpr(name, expr)
		if err != nil {
			return nil, err
		}

		return ast.Any[string]{Value: expr}, nil
	case external.BoolLiteral:
		expr, err := ConvertBoolExpr(name, expr)
		if err != nil {
			return nil, err
		}

		return ast.Any[bool]{Value: expr}, nil
	case external.MapCollection:
		expr, err := ConvertMapExpr(name, expr)
		if err != nil {
			return nil, err
		}

		return ast.Any[map[string]any]{Value: expr}, nil
	case external.Resource:
		expr, err := ConvertResourceExpr(name, expr)
		if err != nil {
			return nil, err
		}

		return ast.Any[state.Resource]{Value: expr}, nil
	default:
		return nil, fmt.Errorf("invalid expr: %T", expr.Value)
	}
}

func ConvertStringExpr(name string, expr external.Expr) (ast.Expr[string], error) {
	switch value := expr.Value.(type) {
	case external.StringLiteral:
		return ast.Literal[string]{Value: value.Value}, nil
	default:
		return nil, fmt.Errorf("invalid string expr: %T", expr)
	}
}

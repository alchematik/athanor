package convert

import (
	"fmt"
	"log/slog"

	external "github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/state"
)

type Converter struct {
	Logger               *slog.Logger
	BlueprintInterpreter BlueprintInterpreter
}

type BlueprintInterpreter interface {
	InterpretBlueprint(source external.BlueprintSource, input map[string]any) (external.Blueprint, error)
}

func ConvertBoolExpr[T any | bool](name string, expr external.Expr) (ast.Expr[T], error) {
	switch value := expr.Value.(type) {
	case external.BoolLiteral:
		var val T
		switch v := any(&value).(type) {
		case *bool:
			*v = value.Value
		case *any:
			*v = value.Value
		}
		return ast.Literal[T]{Value: val}, nil
	default:
		return nil, fmt.Errorf("invalid bool expr: %T", expr)
	}
}

func ConvertMapExpr[T any | map[string]any](name string, expr external.Expr) (ast.Expr[T], error) {
	switch value := expr.Value.(type) {
	case external.MapCollection:
		m := ast.Map[T]{
			Value: map[ast.Expr[string]]ast.Expr[any]{},
		}
		for k, v := range value.Value {
			key, err := ConvertStringExpr[string](name, external.Expr{Value: external.StringLiteral{Value: k}})
			if err != nil {
				return nil, err
			}

			val, err := ConvertAnyExpr(name, v)
			if err != nil {
				return nil, err
			}

			m.Value[key] = val
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

		exists, err := ConvertBoolExpr[bool](name, value.Exists)
		if err != nil {
			return nil, err
		}

		return ast.ResourceExpr[state.Resource]{
			Name:       name,
			Identifier: identifier,
			Config:     config,
			Exists:     exists,
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

func (c *Converter) ConvertStmt(s *state.State, g *ast.Global, stmt external.Stmt) (any, error) {
	switch stmt := stmt.Value.(type) {
	case external.DeclareBuild:
		return c.ConvertBuildStmt(s, g, stmt)
	case external.DeclareResource:
		return c.ConvertResourceStmt(s, g, stmt)
	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

func (c *Converter) ConvertResourceStmt(s *state.State, g *ast.Global, stmt external.DeclareResource) (ast.StmtResource, error) {
	resource, err := ConvertResourceExpr(stmt.Name, stmt.Resource)
	if err != nil {
		return ast.StmtResource{}, err
	}

	resourceID := g.ComponentID(stmt.Name)
	r := ast.StmtResource{
		ID:       resourceID,
		Name:     stmt.Name,
		Resource: resource,
	}

	// g.SetEvaluable(resourceID, r)
	g.SetResource(resourceID, r)
	s.Resources[resourceID] = &state.ResourceState{
		Resource: state.Resource{
			Name: stmt.Name,
		},
	}
	return r, nil
}

func (c *Converter) ConvertBuildStmt(s *state.State, g *ast.Global, build external.DeclareBuild) (ast.StmtBuild, error) {
	blueprint, err := c.BlueprintInterpreter.InterpretBlueprint(build.BlueprintSource, build.Input)
	if err != nil {
		return ast.StmtBuild{}, err
	}

	runtimeInput, err := ConvertMapExpr[map[string]any](build.Name, build.Runtimeinput)
	if err != nil {
		return ast.StmtBuild{}, fmt.Errorf("converting runtime input: %s", err)
	}

	buildID := g.ComponentID(build.Name)

	var stmts []any
	subG := g.Sub(build.Name)
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
		RuntimeInput: runtimeInput,
		Stmts:        stmts,
	}

	g.SetEvaluable(buildID, b)
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
		return ConvertStringExpr[any](name, expr)
	case external.BoolLiteral:
		return ConvertBoolExpr[any](name, expr)
	// case external.MapCollection:
	// 	return ConvertMapExpr(scope, name, expr)
	// case external.Resource:
	// 	return ConvertResourceExpr(scope, name, expr)
	// case external.LocalFile:
	// return ConvertFileExpr(expr)
	case external.MapCollection:
		return ConvertMapExpr[any](name, expr)
	case external.Resource:
		return ast.ResourceExpr[any]{
			// Exists:

		}, nil
	default:
		return nil, fmt.Errorf("invalid expr: %T", expr.Value)
	}
}

func ConvertStringExpr[T any | string](name string, expr external.Expr) (ast.Expr[T], error) {
	switch value := expr.Value.(type) {
	case external.StringLiteral:
		var val T
		switch v := any(&val).(type) {
		case *string:
			*v = value.Value
		case *any:
			*v = value.Value
		default:
			return nil, fmt.Errorf("unsupported string type: %T", val)
		}
		return ast.Literal[T]{Value: val}, nil
	default:
		return nil, fmt.Errorf("invalid bool expr: %T", expr)
	}
}

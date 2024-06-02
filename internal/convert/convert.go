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

func ConvertBoolExpr[T any | bool](scope *ast.Scope, name string, expr external.Expr) (ast.Expr[T], error) {
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

func ConvertMapExpr[T any | map[string]any](scope *ast.Scope, name string, expr external.Expr) (ast.Expr[T], error) {
	switch value := expr.Value.(type) {
	case external.MapCollection:
		m := ast.Map[T]{
			Value: map[ast.Expr[string]]ast.Expr[any]{},
		}
		for k, v := range value.Value {
			key, err := ConvertStringExpr[string](scope, name, external.Expr{Value: external.StringLiteral{Value: k}})
			if err != nil {
				return nil, err
			}

			val, err := ConvertAnyExpr(scope, name, v)
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

func ConvertResourceExpr(scope *ast.Scope, name string, expr external.Expr) (ast.Expr[state.Resource], error) {
	switch value := expr.Value.(type) {
	case external.Resource:
		identifier, err := ConvertAnyExpr(scope, name, value.Identifier)
		if err != nil {
			return nil, err
		}

		config, err := ConvertAnyExpr(scope, name, value.Config)
		if err != nil {
			return nil, err
		}

		exists, err := ConvertBoolExpr[bool](scope, name, value.Exists)
		if err != nil {
			return nil, err
		}

		return ast.ResourceExpr[state.Resource]{
			Identifier: identifier,
			Config:     config,
			Exists:     exists,
		}, nil
	case external.GetResource:
		from, err := ConvertAnyExpr(scope, name, value.From)
		if err != nil {
			return nil, err
		}

		return ast.GetResource{Name: value.Name, From: from}, nil
	default:
		return nil, fmt.Errorf("invalid resource expr: %T", expr)
	}
}

func (c *Converter) ConvertStmt(g *ast.Global, scope *ast.Scope, stmt external.Stmt) (ast.Stmt, error) {
	switch stmt := stmt.Value.(type) {
	case external.DeclareBuild:
		return c.ConvertBuildStmt(g, scope, stmt)
	case external.DeclareResource:
		return c.ConvertResourceStmt(g, scope, stmt)
	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

func (c *Converter) ConvertResourceStmt(g *ast.Global, scope *ast.Scope, stmt external.DeclareResource) (ast.StmtResource, error) {
	resource, err := ConvertResourceExpr(scope, stmt.Name, stmt.Resource)
	if err != nil {
		return ast.StmtResource{}, err
	}

	r := ast.StmtResource{
		Name:     stmt.Name,
		Resource: resource,
	}
	resourceID := g.ComponentID(stmt.Name)
	scope.SetResource(resourceID, stmt.Name)
	g.SetEvaluable(resourceID, r)
	return r, nil
}

func (c *Converter) ConvertBuildStmt(g *ast.Global, scope *ast.Scope, build external.DeclareBuild) (ast.StmtBuild, error) {
	blueprint, err := c.BlueprintInterpreter.InterpretBlueprint(build.BlueprintSource, build.Input)
	if err != nil {
		return ast.StmtBuild{}, err
	}

	runtimeInput, err := ConvertMapExpr[map[string]any](scope, build.Name, build.Runtimeinput)
	if err != nil {
		return ast.StmtBuild{}, fmt.Errorf("converting runtime input: %s", err)
	}

	buildID := g.ComponentID(build.Name)

	var stmts []ast.Stmt
	subScope := scope.SetBuild(buildID, build.Name)
	subG := g.Sub(build.Name)
	for _, stmt := range blueprint.Stmts {
		s, err := c.ConvertStmt(subG, subScope, stmt)
		if err != nil {
			return ast.StmtBuild{}, err
		}

		stmts = append(stmts, s)
	}

	b := ast.StmtBuild{
		Name: build.Name,
		Build: ast.Build{
			RuntimeInput: runtimeInput,
			Blueprint: ast.Blueprint{
				Stmts: stmts,
			},
		},
	}

	g.SetEvaluable(buildID, b)

	return b, nil
}

func ConvertAnyExpr(scope *ast.Scope, name string, expr external.Expr) (ast.Expr[any], error) {
	switch expr.Value.(type) {
	case external.StringLiteral:
		return ConvertStringExpr[any](scope, name, expr)
	case external.BoolLiteral:
		return ConvertBoolExpr[any](scope, name, expr)
	// case external.MapCollection:
	// 	return ConvertMapExpr(scope, name, expr)
	// case external.Resource:
	// 	return ConvertResourceExpr(scope, name, expr)
	// case external.LocalFile:
	// return ConvertFileExpr(expr)
	case external.MapCollection:
		return ConvertMapExpr[any](scope, name, expr)
	case external.Resource:
		return ast.ResourceExpr[any]{
			// Exists:

		}, nil
	default:
		return nil, fmt.Errorf("invalid expr: %T", expr.Value)
	}
}

func ConvertStringExpr[T any | string](scope *ast.Scope, name string, expr external.Expr) (ast.Expr[T], error) {
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

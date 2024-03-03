package interpreter

import (
	"context"
	"fmt"
	"slices"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/spec"
)

func (in Interpreter) Stmt(ctx context.Context, b spec.Spec, st ast.Stmt) error {
	switch s := st.(type) {
	case ast.StmtResource:
		return in.resourceStmt(ctx, b, s)
	case ast.StmtBuild:
		return in.buildStmt(ctx, b, s)
	default:
		return fmt.Errorf("unknown stmt %T", st)
	}
}

func (in Interpreter) buildStmt(ctx context.Context, s spec.Spec, stmt ast.StmtBuild) error {
	runtimeConfig, children, err := in.Expr(ctx, s, stmt.RuntimeConfig)
	if err != nil {
		return err
	}

	subSpec := spec.Spec{
		DependencyMap: map[string][]string{},
		Components:    map[string]spec.Component{},
		RuntimeConfig: runtimeConfig,
	}

	bp, err := in.Translator.Translate(ctx, stmt)
	if err != nil {
		return err
	}

	// TODO: ast.ExprBlueprint vs ast.Blueprint?
	_, _, err = in.Expr(ctx, subSpec, ast.ExprBlueprint{Stmts: bp.Stmts})
	if err != nil {
		return err
	}

	s.DependencyMap[stmt.Alias] = append(s.DependencyMap[stmt.Alias], children...)
	s.Components[stmt.Alias] = spec.ComponentBuild{Spec: subSpec}

	return nil
}

func (in Interpreter) resourceStmt(ctx context.Context, b spec.Spec, s ast.StmtResource) error {
	val, children, err := in.Expr(ctx, b, s.Expr)
	if err != nil {
		return err
	}

	resource, ok := val.(spec.ValueResource)
	if !ok {
		return fmt.Errorf("expected Resource type, got %T", val)
	}

	alias := resource.Identifier.Alias
	b.DependencyMap[alias] = slices.Compact(append(b.DependencyMap[alias], children...))
	b.Components[alias] = spec.ComponentResource{Value: resource}

	return nil
}

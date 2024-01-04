package interpreter

import (
	"context"
	"fmt"
	"slices"

	"github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/build"
	"github.com/alchematik/athanor/build/component"
	"github.com/alchematik/athanor/build/value"
)

func (in Interpreter) Stmt(ctx context.Context, b build.Build, st ast.Stmt) error {
	switch s := st.(type) {
	case ast.StmtProvider:
		return in.providerStmt(ctx, b, s)
	case ast.StmtResource:
		return in.resourceStmt(ctx, b, s)
	default:
		return fmt.Errorf("unknown stmt %T", st)
	}
}

func (in Interpreter) providerStmt(ctx context.Context, b build.Build, s ast.StmtProvider) error {
	val, _, err := in.Expr(ctx, b, s.Expr)
	if err != nil {
		return err
	}

	provider, ok := val.(value.Provider)
	if !ok {
		return fmt.Errorf("expected Provider type, got %T", val)
	}

	b.Providers[provider.Identifier.Alias] = provider

	return nil
}

func (in Interpreter) resourceStmt(ctx context.Context, b build.Build, s ast.StmtResource) error {
	val, children, err := in.Expr(ctx, b, s.Expr)
	if err != nil {
		return err
	}

	resource, ok := val.(value.Resource)
	if !ok {
		return fmt.Errorf("expected Resource type, got %T", val)
	}

	alias := resource.Identifier.Alias
	b.DependencyMap[alias] = slices.Compact(append(b.DependencyMap[alias], children...))
	b.Resources[alias] = resource
	b.Providers[resource.Provider.Identifier.Alias] = resource.Provider
	b.Components[alias] = component.Resource{Value: resource}

	return nil
}

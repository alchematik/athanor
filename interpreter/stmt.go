package interpreter

import (
	"context"
	"fmt"
	"slices"

	"github.com/alchematik/athanor/blueprint/stmt"
	"github.com/alchematik/athanor/build"
	"github.com/alchematik/athanor/build/component"
	"github.com/alchematik/athanor/build/value"
)

func (in Interpreter) Stmt(ctx context.Context, b build.Build, st stmt.Type) error {
	switch s := st.(type) {
	case stmt.Provider:
		return in.providerStmt(ctx, b, s)
	case stmt.Resource:
		return in.resourceStmt(ctx, b, s)
	default:
		return fmt.Errorf("unknown stmt %T", st)
	}
}

func (in Interpreter) providerStmt(ctx context.Context, b build.Build, s stmt.Provider) error {
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

func (in Interpreter) resourceStmt(ctx context.Context, b build.Build, s stmt.Resource) error {
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

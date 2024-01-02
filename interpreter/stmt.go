package interpreter

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/blueprint/stmt"
	"github.com/alchematik/athanor/build/value"
)

func (in Interpreter) Stmt(ctx context.Context, build value.Build, st stmt.Type) error {
	switch s := st.(type) {
	case stmt.Provider:
		return in.providerStmt(ctx, build, s)
	case stmt.Resource:
		return in.resourceStmt(ctx, build, s)
	default:
		return fmt.Errorf("unknown stmt %T", st)
	}
}

func (in Interpreter) providerStmt(ctx context.Context, build value.Build, s stmt.Provider) error {
	val, _, err := in.Expr(ctx, build, s.Expr)
	if err != nil {
		return err
	}

	provider, ok := val.(value.Provider)
	if !ok {
		return fmt.Errorf("expected Provider type, got %T", val)
	}

	build.Providers[provider.Identifier.Alias] = provider

	return nil
}

func (in Interpreter) resourceStmt(ctx context.Context, build value.Build, s stmt.Resource) error {
	val, children, err := in.Expr(ctx, build, s.Expr)
	if err != nil {
		return err
	}

	resource, ok := val.(value.Resource)
	if !ok {
		return fmt.Errorf("expected Resource type, got %T", val)
	}

	alias := resource.Identifier.Alias
	build.DependencyMap[alias] = append(build.DependencyMap[alias], children...)
	build.Resources[alias] = resource
	build.Providers[resource.Provider.Identifier.Alias] = resource.Provider

	return nil
}

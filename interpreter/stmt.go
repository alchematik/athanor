package interpreter

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/blueprint/stmt"
	"github.com/alchematik/athanor/build/value"
)

func (in Interpreter) Stmt(ctx context.Context, env Environment, st stmt.Type) error {
	switch s := st.(type) {
	case stmt.Provider:
		return in.providerStmt(ctx, env, s)
	case stmt.Resource:
		return in.resourceStmt(ctx, env, s)
	default:
		return fmt.Errorf("unknown stmt %T", st)
	}
}

func (in Interpreter) providerStmt(ctx context.Context, env Environment, s stmt.Provider) error {
	val, _, err := in.Expr(ctx, env, s.Expr)
	if err != nil {
		return err
	}

	provider, ok := val.(value.Provider)
	if !ok {
		return fmt.Errorf("expected Provider type, got %T", val)
	}

	env.Providers[provider.Identifier.Alias] = provider

	return nil
}

func (in Interpreter) resourceStmt(ctx context.Context, env Environment, s stmt.Resource) error {
	val, children, err := in.Expr(ctx, env, s.Expr)
	if err != nil {
		return err
	}

	resource, ok := val.(value.Resource)
	if !ok {
		return fmt.Errorf("expected Resource type, got %T", val)
	}

	alias := resource.Identifier.Alias
	env.DependencyMap[alias] = append(env.DependencyMap[alias], children...)
	env.Resources[alias] = resource
	env.Providers[resource.Provider.Identifier.Alias] = resource.Provider

	return nil
}

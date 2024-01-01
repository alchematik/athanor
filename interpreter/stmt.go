package interpreter

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/blueprint/stmt"
	"github.com/alchematik/athanor/build/value"
)

func (in Interpreter) ProviderStmt(ctx context.Context, env Environment, s stmt.Provider) error {
	id, _, err := in.Expr(ctx, env, s.Identifier)
	if err != nil {
		return err
	}

	providerID, ok := id.(value.ProviderIdentifier)
	if !ok {
		return fmt.Errorf("expected Provider type, got %T", id)
	}

	env.Providers[providerID.Alias] = value.Provider{
		Identifier: providerID,
	}

	return nil
}

func (in Interpreter) ResourceStmt(ctx context.Context, env Environment, s stmt.Resource) error {
	providerValue, _, err := in.Expr(ctx, env, s.Provider)
	if err != nil {
		return err
	}

	provider, ok := providerValue.(value.Provider)
	if !ok {
		return fmt.Errorf("expected Provider type, got %T", providerValue)
	}

	identifierValue, identifierChildren, err := in.Expr(ctx, env, s.Identifier)
	if err != nil {
		return err
	}

	identifier, ok := identifierValue.(value.ResourceIdentifier)
	if !ok {
		return fmt.Errorf("expectedesesource type, got %T", identifierValue)
	}

	config, configChildren, err := in.Expr(ctx, env, s.Config)
	if err != nil {
		return err
	}

	var children []string
	for _, child := range append(identifierChildren, configChildren...) {
		if child == identifier.Alias {
			continue
		}

		children = append(children, child)
	}

	env.DependencyMap[identifier.Alias] = append(env.DependencyMap[identifier.Alias], children...)
	env.Resources[identifier.Alias] = value.Resource{
		Provider:   provider,
		Identifier: identifier,
		Config:     config,
		Attrs: value.Unresolved{
			Name: "attrs",
			Object: value.ResourceRef{
				Alias: identifier.Alias,
			},
		},
	}
	return nil
}

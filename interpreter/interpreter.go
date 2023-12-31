package interpreter

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/blueprint"
	"github.com/alchematik/athanor/blueprint/expr"
	"github.com/alchematik/athanor/blueprint/stmt"
	"github.com/alchematik/athanor/build/value"
)

// Stmts and expressions

type Interpreter struct {
	ResourcesAPI ResourcesAPI
}

type Environment struct {
	Providers map[string]value.Provider
	Resources map[string]value.Resource

	// TODO: construct dependency map
	DependencyMap map[string][]string
}

type ResourcesAPI interface {
	FetchResource(ctx context.Context, r value.Resource) (value.Resource, error)
}

type NilResourcesAPI struct {
}

func (api NilResourcesAPI) FetchResource(ctx context.Context, r value.Resource) (value.Resource, error) {
	r.Attrs = value.Unresolved{
		Name:   "",
		Object: r,
	}
	return r, nil
}

func (in Interpreter) Interpret(ctx context.Context, env Environment, b blueprint.Blueprint) error {
	for _, st := range b.Stmts {
		fmt.Printf("stmt: %T\n", st)
		switch s := st.(type) {
		case stmt.Provider:
			// TODO: Use dependencies.
			ex, _, err := in.InterpretExpr(ctx, env, s.Identifier)
			if err != nil {
				return err
			}

			provider, ok := ex.(value.Provider)
			if !ok {
				return fmt.Errorf("expected Provider type, got %T", ex)
			}

			env.Providers[provider.Alias] = provider
		case stmt.Resource:
			providerValue, _, err := in.InterpretExpr(ctx, env, s.Provider)
			if err != nil {
				return err
			}

			provider, ok := providerValue.(value.Provider)
			if !ok {
				return fmt.Errorf("expected Provider type, got %T", providerValue)
			}

			identifierValue, identifierChildren, err := in.InterpretExpr(ctx, env, s.Identifier)
			if err != nil {
				return err
			}

			identifier, ok := identifierValue.(value.ResourceIdentifier)
			if !ok {
				return fmt.Errorf("expectedesesource type, got %T", identifierValue)
			}

			config, configChildren, err := in.InterpretExpr(ctx, env, s.Config)
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
		default:
			return fmt.Errorf("unknown stmt %T", st)
		}
	}

	return nil
}

func (in Interpreter) InterpretExpr(ctx context.Context, env Environment, ex expr.Type) (value.Type, []string, error) {
	switch e := ex.(type) {
	case expr.String:
		return value.String{Value: e.Value}, nil, nil
	case expr.Map:
		m := value.Map{Entries: map[string]value.Type{}}
		var children []string
		for k, v := range e.Entries {
			var err error
			var valChildren []string

			fmt.Printf("map: %v -> %T\n", k, v)
			m.Entries[k], valChildren, err = in.InterpretExpr(ctx, env, v)
			if err != nil {
				return nil, nil, err
			}

			children = append(children, valChildren...)
		}

		return m, children, nil
	case expr.ProviderIdentifier:
		name, nameChildren, err := in.InterpretExpr(ctx, env, e.Name)
		if err != nil {
			return nil, nil, err
		}

		nameStr, ok := name.(value.String)
		if !ok {
			return nil, nil, fmt.Errorf("provider name must be a string")
		}

		version, versionChildren, err := in.InterpretExpr(ctx, env, e.Version)
		if err != nil {
			return nil, nil, err
		}

		versionStr, ok := version.(value.String)
		if !ok {
			return nil, nil, fmt.Errorf("provider version must be a string")
		}

		children := append(nameChildren, versionChildren...)

		return value.Provider{
			Alias:   e.Alias,
			Name:    nameStr.Value,
			Version: versionStr.Value,
		}, children, nil
	case expr.ResourceIdentifier:
		val, children, err := in.InterpretExpr(ctx, env, e.Value)
		if err != nil {
			return nil, nil, err
		}

		return value.ResourceIdentifier{
			Alias:        e.Alias,
			ResourceType: e.ResourceType,
			Value:        val,
		}, append(children, e.Alias), nil
	case expr.IOGet:
		objVal, children, err := in.InterpretExpr(ctx, env, e.Object)
		if err != nil {
			return nil, nil, err
		}

		unresolved, ok := objVal.(value.Unresolved)
		if !ok {
			return nil, nil, fmt.Errorf("property %q does not belong to unresolved object; use get", e.Name)
		}

		return value.Unresolved{
			Name:   e.Name,
			Object: unresolved,
		}, children, nil
	case expr.Get:
		var m map[string]value.Type

		objVal, children, err := in.InterpretExpr(ctx, env, e.Object)
		if err != nil {
			return nil, nil, err
		}

		switch obj := objVal.(type) {
		case value.Map:
			m = obj.Entries
		case value.Resource:
			m = map[string]value.Type{
				"identifier": obj.Identifier,
				"config":     obj.Config,
				"attrs":      obj.Attrs,
			}
		case value.Unresolved:
			return nil, nil, fmt.Errorf("property %q belongs to an unresolved object; use io_get", e.Name)
		default:
			return nil, nil, fmt.Errorf("cannot access property %q", e.Name)
		}

		val, ok := m[e.Name]
		if !ok {
			return nil, nil, fmt.Errorf("property %q not set", e.Name)
		}

		return val, children, nil
	case expr.GetProvider:
		p, ok := env.Providers[e.Alias]
		if !ok {
			return nil, nil, fmt.Errorf("provider with alias %q does not exist", e.Alias)
		}

		return p, nil, nil
	case expr.GetResource:
		r, ok := env.Resources[e.Alias]
		if !ok {
			return nil, nil, fmt.Errorf("resource with alias %q does not exist", e.Alias)
		}

		return r, nil, nil
	default:
		return nil, nil, fmt.Errorf("unknown expr %T", ex)
	}
}

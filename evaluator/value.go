package evaluator

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/build/value"
	"github.com/alchematik/athanor/state"
)

func provider(v value.Provider) (state.Provider, error) {
	if v.Identifier.Name == "" {
		return state.Provider{}, fmt.Errorf("name is required for provider")
	}
	if v.Identifier.Version == "" {
		return state.Provider{}, fmt.Errorf("version is required for provider")
	}
	return state.Provider{
		Name:    v.Identifier.Name,
		Version: v.Identifier.Version,
	}, nil
}

func (e Evaluator) Value(ctx context.Context, env state.Environment, val value.Type) (state.Type, error) {
	switch v := val.(type) {
	case value.Provider:
		return provider(v)
	case value.Resource:
		idState, err := e.Value(ctx, env, v.Identifier)
		if err != nil {
			return nil, err
		}

		id, ok := idState.(state.Identifier)
		if !ok {
			return nil, fmt.Errorf("expected Identifier, got %T", idState)
		}

		config, err := e.Value(ctx, env, v.Config)
		if err != nil {
			return nil, err
		}

		providerState, err := e.Value(ctx, env, v.Provider)
		if err != nil {
			return nil, err
		}

		provider, ok := providerState.(state.Provider)
		if !ok {
			return nil, fmt.Errorf("expected Provider, got %T", providerState)
		}

		input := state.Resource{
			Provider:   provider,
			Identifier: id,
			Config:     config,
		}

		output, err := e.ResourceAPI.GetResource(ctx, input)
		if err != nil {
			return nil, err
		}

		return output, nil
	case value.String:
		return state.String{Value: v.Value}, nil
	case value.Map:
		m := state.Map{
			Entries: map[string]state.Type{},
		}

		for k, entry := range v.Entries {
			resolved, err := e.Value(ctx, env, entry)
			if err != nil {
				return nil, err
			}

			m.Entries[k] = resolved
		}

		return m, nil
	case value.ResourceIdentifier:
		val, err := e.Value(ctx, env, v.Value)
		if err != nil {
			return nil, err
		}

		return state.Identifier{
			Alias:        v.Alias,
			ResourceType: v.ResourceType,
			Value:        val,
		}, nil
	case value.ResourceRef:
		r, ok := env.Resources[v.Alias]
		if !ok {
			return nil, fmt.Errorf("evaluator: resource with alias %q does not exist", v.Alias)
		}

		return r, nil
	case value.Unresolved:
		if _, ok := v.Object.(value.Nil); ok {
			obj, inEnv := env.Resources[v.Name]
			if !inEnv {
				return nil, fmt.Errorf("object %q not in env", v.Name)
			}

			return obj, nil
		}

		resolved, err := e.Value(ctx, env, v.Object)
		if err != nil {
			return nil, err
		}

		var m map[string]state.Type
		switch obj := resolved.(type) {
		case state.Resource:
			m = map[string]state.Type{
				"identifier": obj.Identifier,
				"config":     obj.Config,
				"attrs":      obj.Attrs,
			}
		case state.Unknown:
			return state.Unknown{
				Name:   v.Name,
				Object: resolved,
			}, nil
		case state.Map:
			m = obj.Entries
		default:
			return nil, fmt.Errorf("value type %T has no field %q", v.Object, v.Name)
		}

		field, ok := m[v.Name]
		if !ok {
			return nil, fmt.Errorf("property %q not set", v.Name)
		}

		return field, nil
	default:
		return nil, fmt.Errorf("unrecognized value type: %T", val)
	}
}

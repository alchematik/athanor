package evaluator

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/build/value"
	"github.com/alchematik/athanor/state"
)

func (e Evaluator) providerValue(v value.Provider) (state.Provider, error) {
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

func (e Evaluator) mapValue(ctx context.Context, env state.Environment, v value.Map) (state.Map, error) {
	m := state.Map{
		Entries: map[string]state.Type{},
	}

	for k, entry := range v.Entries {
		resolved, err := e.Value(ctx, env, entry)
		if err != nil {
			return state.Map{}, err
		}

		m.Entries[k] = resolved
	}

	return m, nil
}

func (e Evaluator) resourceIdentifier(ctx context.Context, env state.Environment, v value.ResourceIdentifier) (state.Identifier, error) {
	if v.Alias == "" {
		return state.Identifier{}, fmt.Errorf("alias is required")
	}

	if v.ResourceType == "" {
		return state.Identifier{}, fmt.Errorf("resource type is required")
	}

	val, err := e.Value(ctx, env, v.Value)
	if err != nil {
		return state.Identifier{}, err
	}

	return state.Identifier{
		Alias:        v.Alias,
		ResourceType: v.ResourceType,
		Value:        val,
	}, nil
}

func (e Evaluator) resourceRef(env state.Environment, v value.ResourceRef) (state.Resource, error) {
	r, ok := env.Resources[v.Alias]
	if !ok {
		return state.Resource{}, fmt.Errorf("evaluator: resource with alias %q does not exist", v.Alias)
	}

	return r, nil
}

func (e Evaluator) unresolvedValue(ctx context.Context, env state.Environment, v value.Unresolved) (state.Type, error) {
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
		return nil, fmt.Errorf("value type %T has no field %q", resolved, v.Name)
	}

	field, ok := m[v.Name]
	if !ok {
		return nil, fmt.Errorf("property %q not set", v.Name)
	}

	return field, nil
}

func (e Evaluator) resourceValue(ctx context.Context, env state.Environment, v value.Resource) (state.Resource, error) {
	idState, err := e.Value(ctx, env, v.Identifier)
	if err != nil {
		return state.Resource{}, err
	}

	id, ok := idState.(state.Identifier)
	if !ok {
		return state.Resource{}, fmt.Errorf("expected Identifier, got %T", idState)
	}

	config, err := e.Value(ctx, env, v.Config)
	if err != nil {
		return state.Resource{}, err
	}

	providerState, err := e.Value(ctx, env, v.Provider)
	if err != nil {
		return state.Resource{}, err
	}

	provider, ok := providerState.(state.Provider)
	if !ok {
		return state.Resource{}, fmt.Errorf("expected Provider, got %T", providerState)
	}

	existsVal, err := e.Value(ctx, env, v.Exists)
	if err != nil {
		return state.Resource{}, err
	}

	exists, ok := existsVal.(state.Bool)
	if !ok {
		return state.Resource{}, fmt.Errorf("exists must be boolean")
	}

	// fmt.Printf("exists >>>>>>>>>>> %v\n", exists)

	input := state.Resource{
		Provider:   provider,
		Identifier: id,
		Config:     config,
		Exists:     exists,
	}

	output, err := e.ResourceAPI.GetResource(ctx, input)
	if err != nil {
		return state.Resource{}, err
	}

	return output, nil
}

func (e Evaluator) Value(ctx context.Context, env state.Environment, val value.Type) (state.Type, error) {
	// fmt.Printf(">>>>>>>>>>> %T, %v\n", val, val)
	switch v := val.(type) {
	case value.Provider:
		return e.providerValue(v)
	case value.Resource:
		return e.resourceValue(ctx, env, v)
	case value.String:
		return state.String{Value: v.Value}, nil
	case value.Bool:
		return state.Bool{Value: v.Value}, nil
	case value.Map:
		return e.mapValue(ctx, env, v)
	case value.ResourceIdentifier:
		return e.resourceIdentifier(ctx, env, v)
	case value.ResourceRef:
		return e.resourceRef(env, v)
	case value.Unresolved:
		return e.unresolvedValue(ctx, env, v)
	default:
		return nil, fmt.Errorf("unrecognized value type: %T", val)
	}
}

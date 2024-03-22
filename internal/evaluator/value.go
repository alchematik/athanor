package evaluator

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"os"

	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"
)

func (e Evaluator) providerValue(v spec.ValueProvider) (state.Provider, error) {
	return state.Provider{
		Repo: v.Repo,
	}, nil
}

func (e Evaluator) mapValue(ctx context.Context, env state.Environment, v spec.ValueMap) (state.Map, error) {
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

func (e Evaluator) listValue(ctx context.Context, env state.Environment, v spec.ValueList) (state.List, error) {
	l := state.List{
		Elements: make([]state.Type, len(v.Elements)),
	}
	for i, val := range v.Elements {
		resolved, err := e.Value(ctx, env, val)
		if err != nil {
			return state.List{}, err
		}

		l.Elements[i] = resolved
	}

	return l, nil
}

func (e Evaluator) fileValue(ctx context.Context, env state.Environment, f spec.ValueFile) (state.File, error) {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state.File{}, nil
		}

		return state.File{}, err
	}

	checksum := crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli))

	return state.File{
		Path:     f.Path,
		Checksum: fmt.Sprintf("%d", checksum),
	}, nil
}

func (e Evaluator) resourceIdentifier(ctx context.Context, env state.Environment, v spec.ValueResourceIdentifier) (state.Identifier, error) {
	if v.Alias == "" {
		return state.Identifier{}, fmt.Errorf("alias is required")
	}

	if v.ResourceType == "" {
		return state.Identifier{}, fmt.Errorf("resource type is required")
	}

	val, err := e.Value(ctx, env, v.Literal)
	if err != nil {
		return state.Identifier{}, err
	}

	return state.Identifier{
		Alias:        v.Alias,
		ResourceType: v.ResourceType,
		Value:        val,
	}, nil
}

func (e Evaluator) resourceRef(env state.Environment, v spec.ValueResourceRef) (state.Resource, error) {
	r, ok := env.States[v.Alias]
	if !ok {
		return state.Resource{}, fmt.Errorf("evaluator: resource with alias %q does not exist", v.Alias)
	}

	res, ok := r.(state.Resource)
	if !ok {
		return state.Resource{}, fmt.Errorf("expected Resource type, got %T", r)
	}

	return res, nil
}

func (e Evaluator) unresolvedValue(ctx context.Context, env state.Environment, v spec.ValueUnresolved) (state.Type, error) {
	resolved, err := e.Value(ctx, env, v.Object)
	if err != nil {
		return nil, err
	}

	var m map[string]state.Type
	switch obj := resolved.(type) {
	case state.Nil:
		s, ok := env.States[v.Name]
		if !ok {
			return nil, fmt.Errorf("value with alias %q not found", v.Name)
		}

		return s, nil
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

func (e Evaluator) resourceValue(ctx context.Context, env state.Environment, v spec.ValueResource) (state.Resource, error) {
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

func (e Evaluator) Value(ctx context.Context, env state.Environment, val spec.Value) (state.Type, error) {
	switch v := val.(type) {
	case spec.ValueProvider:
		return e.providerValue(v)
	case spec.ValueResource:
		return e.resourceValue(ctx, env, v)
	case spec.ValueString:
		return state.String{Value: v.Literal}, nil
	case spec.ValueBool:
		return state.Bool{Value: v.Literal}, nil
	case spec.ValueMap:
		return e.mapValue(ctx, env, v)
	case spec.ValueList:
		return e.listValue(ctx, env, v)
	case spec.ValueResourceIdentifier:
		return e.resourceIdentifier(ctx, env, v)
	case spec.ValueResourceRef:
		return e.resourceRef(env, v)
	case spec.ValueUnresolved:
		return e.unresolvedValue(ctx, env, v)
	case spec.ValueFile:
		return e.fileValue(ctx, env, v)
	case spec.ValueRuntimeConfig:
		return state.Unknown{
			Object: state.RuntimeConfig{},
		}, nil
	case spec.ValueNil:
		return state.Nil{}, nil
	default:
		return nil, fmt.Errorf("unrecognized value type: %T", val)
	}
}

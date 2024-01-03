package evaluator

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/build/value"
	"github.com/alchematik/athanor/state"
)

type Evaluator struct {
	ResourceAPI ResourceAPI
}

type ResourceAPI interface {
	GetResource(context.Context, state.Resource) (state.Resource, error)
}

func (e Evaluator) Evaluate(ctx context.Context, env state.Environment, val value.Type) (state.Type, error) {
	switch v := val.(type) {
	case value.Provider:
		return state.Provider{
			Name:    v.Identifier.Name,
			Version: v.Identifier.Version,
		}, nil
	case value.Resource:
		idState, err := e.Evaluate(ctx, env, v.Identifier)
		if err != nil {
			return nil, err
		}

		id, ok := idState.(state.Identifier)
		if !ok {
			return nil, fmt.Errorf("expected Identifier, got %T", idState)
		}

		config, err := e.Evaluate(ctx, env, v.Config)
		if err != nil {
			return nil, err
		}

		providerState, err := e.Evaluate(ctx, env, v.Provider)
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

		env.Resources[input.Identifier.Alias] = output

		return output, nil
	case value.Build:
		indegrees := map[string]int{}
		parentToChildren := map[string][]string{}
		for child, parents := range v.DependencyMap {
			indegrees[child] = len(parents)
			for _, parent := range parents {
				parentToChildren[parent] = append(parentToChildren[parent], child)
			}
		}

		// TODO: detect cycle.

		var queue []string
		for alias, in := range indegrees {
			if in == 0 {
				queue = append(queue, alias)
				delete(indegrees, alias)
			}
		}

		stateEnv := state.Environment{
			DependencyMap: env.DependencyMap,
			Resources:     map[string]state.Resource{},
		}

		// TODO: parallelize.
		for len(queue) > 0 {
			var alias string
			alias, queue = queue[0], queue[1:]

			fmt.Printf("evaluating: %q\n", alias)

			s, err := e.Evaluate(ctx, stateEnv, v.Resources[alias])
			if err != nil {
				return state.Environment{}, err
			}

			stateEnv.Resources[alias] = s.(state.Resource)

			for _, childAlias := range parentToChildren[alias] {
				indegrees[childAlias]--
				if indegrees[childAlias] == 0 {
					queue = append(queue, childAlias)
					delete(indegrees, childAlias)
				}
			}
		}

		return stateEnv, nil
	case value.String:
		return state.String{Value: v.Value}, nil
	case value.Map:
		m := state.Map{
			Entries: map[string]state.Type{},
		}

		for k, entry := range v.Entries {
			resolved, err := e.Evaluate(ctx, env, entry)
			if err != nil {
				return nil, err
			}

			m.Entries[k] = resolved
		}

		return m, nil
	case value.ResourceIdentifier:
		val, err := e.Evaluate(ctx, env, v.Value)
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

		resolved, err := e.Evaluate(ctx, env, v.Object)
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

package evaluator

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/alchematik/athanor/backend"
	"github.com/alchematik/athanor/build/value"
	"github.com/alchematik/athanor/interpreter"
	"github.com/alchematik/athanor/state"
	// TODO: This has to be not internal anymore?
	backendpb "github.com/alchematik/athanor/internal/gen/go/proto/backend/v1"
	statepb "github.com/alchematik/athanor/internal/gen/go/proto/state/v1"

	plugin "github.com/hashicorp/go-plugin"
)

type Evaluator struct {
	ResourceEvaluator ResourceEvaluator
}

type ResourceEvaluator interface {
	// TODO: Should alias be part of the resource struct?
	EvaluateResource(context.Context, state.Environment, string, value.Resource) (state.Resource, error)
}

func (e Evaluator) Evaluate(ctx context.Context, env interpreter.Environment) (state.Environment, error) {
	indegrees := map[string]int{}
	parentToChildren := map[string][]string{}
	for child, parents := range env.DependencyMap {
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

		s, err := e.ResourceEvaluator.EvaluateResource(ctx, stateEnv, alias, env.Resources[alias])
		if err != nil {
			return state.Environment{}, err
		}

		stateEnv.Resources[alias] = s

		for _, childAlias := range parentToChildren[alias] {
			indegrees[childAlias]--
			if indegrees[childAlias] == 0 {
				queue = append(queue, childAlias)
				delete(indegrees, childAlias)
			}
		}
	}

	return stateEnv, nil
}

type ValueResolver interface {
	ResolveValue(context.Context, state.Environment, value.Type) (state.Type, error)
	ResolveResourceIdentifierValue(context.Context, state.Environment, value.ResourceIdentifier) (state.Identifier, error)
}

type PlanResourceEvaluator struct {
	ValueResolver ValueResolver
}

func (e PlanResourceEvaluator) EvaluateResource(ctx context.Context, env state.Environment, alias string, r value.Resource) (state.Resource, error) {
	id, err := e.ValueResolver.ResolveResourceIdentifierValue(ctx, env, r.Identifier)
	if err != nil {
		return state.Resource{}, err
	}

	config, err := e.ValueResolver.ResolveValue(ctx, env, r.Config)
	if err != nil {
		return state.Resource{}, err
	}

	resource := state.Resource{
		Provider: state.Provider{
			Name:    r.Provider.Identifier.Name,
			Version: r.Provider.Identifier.Version,
		},
		Identifier: id,
		Config:     config,
		Attrs: state.Unknown{
			Name: "attrs",
			Object: state.ResourceRef{
				Alias: alias,
			},
		},
	}

	env.Resources[alias] = resource

	return resource, nil
}

type RemoteResourceEvaluator struct {
	ProviderPluginDir string
	ValueResolver     ValueResolver
}

func (e RemoteResourceEvaluator) EvaluateResource(ctx context.Context, env state.Environment, alias string, r value.Resource) (state.Resource, error) {
	id, err := e.ValueResolver.ResolveResourceIdentifierValue(ctx, env, r.Identifier)
	if err != nil {
		return state.Resource{}, err
	}

	pluginPath := filepath.Join(e.ProviderPluginDir, r.Provider.Identifier.Name, r.Provider.Identifier.Version, "provider")
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: backend.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"backend": &backend.Plugin{},
		},
		Cmd:              exec.Command("sh", "-c", pluginPath),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
	})

	dispensor, err := client.Client()
	if err != nil {
		return state.Resource{}, err
	}

	rawPlug, err := dispensor.Dispense("backend")
	if err != nil {
		return state.Resource{}, err
	}

	plug, ok := rawPlug.(backendpb.BackendClient)
	if !ok {
		return state.Resource{}, fmt.Errorf("expected BackendClient, got %T", rawPlug)
	}

	protoID, err := convertProtoValue(id)
	if err != nil {
		return state.Resource{}, err
	}

	res, err := plug.GetResource(ctx, &backendpb.GetResourceRequest{
		Identifier: protoID.GetIdentifier(),
	})
	if err != nil {
		return state.Resource{}, err
	}

	config, err := protoToState(res.GetResource().GetConfig())
	if err != nil {
		return state.Resource{}, err
	}

	attrs, err := protoToState(res.GetResource().GetAttrs())
	if err != nil {
		return state.Resource{}, err
	}

	resource := state.Resource{
		Provider: state.Provider{
			Name:    r.Provider.Identifier.Name,
			Version: r.Provider.Identifier.Version,
		},
		Identifier: id,
		Config:     config,
		Attrs:      attrs,
	}

	env.Resources[alias] = resource
	fmt.Printf("setting alias: %v\n", alias)

	return resource, nil
}

func protoToState(val *statepb.Value) (state.Type, error) {
	switch v := val.GetType().(type) {
	case *statepb.Value_Map:
		entries := map[string]state.Type{}
		for k, element := range v.Map.GetEntries() {
			converted, err := protoToState(element)
			if err != nil {
				return nil, err
			}
			entries[k] = converted
		}

		return state.Map{Entries: entries}, nil
	case *statepb.Value_StringValue:
		return state.String{Value: v.StringValue}, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", val.GetType())
	}
}

func convertProtoValue(val state.Type) (*statepb.Value, error) {
	switch v := val.(type) {
	case state.String:
		return &statepb.Value{
			Type: &statepb.Value_StringValue{StringValue: v.Value},
		}, nil
	case state.Map:
		entries := map[string]*statepb.Value{}
		for k, v := range v.Entries {
			converted, err := convertProtoValue(v)
			if err != nil {
				return nil, err
			}
			entries[k] = converted
		}

		return &statepb.Value{
			Type: &statepb.Value_Map{
				Map: &statepb.MapValue{
					Entries: entries,
				},
			},
		}, nil
	case state.Identifier:
		converted, err := convertProtoValue(v.Value)
		if err != nil {
			return nil, err
		}

		return &statepb.Value{
			Type: &statepb.Value_Identifier{
				Identifier: &statepb.Identifier{
					Type:  v.ResourceType,
					Value: converted,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("convert proto: unknown type %T\n", val)
	}

}

type RealValueResolver struct {
}

func (e RealValueResolver) ResolveResourceIdentifierValue(ctx context.Context, env state.Environment, id value.ResourceIdentifier) (state.Identifier, error) {
	val, err := e.ResolveValue(ctx, env, id.Value)
	if err != nil {
		return state.Identifier{}, err
	}

	return state.Identifier{
		ResourceType: id.ResourceType,
		Value:        val,
	}, nil
}

func (e RealValueResolver) ResolveValue(ctx context.Context, env state.Environment, val value.Type) (state.Type, error) {
	switch v := val.(type) {
	case value.String:
		return state.String{Value: v.Value}, nil
	case value.Map:
		m := state.Map{
			Entries: map[string]state.Type{},
		}

		for k, entry := range v.Entries {
			resolved, err := e.ResolveValue(ctx, env, entry)
			if err != nil {
				return nil, err
			}

			m.Entries[k] = resolved
		}

		return m, nil
	case value.ResourceIdentifier:
		val, err := e.ResolveValue(ctx, env, v.Value)
		if err != nil {
			return nil, err
		}

		return state.Identifier{
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

		resolved, err := e.ResolveValue(ctx, env, v.Object)
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

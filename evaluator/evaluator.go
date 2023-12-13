package evaluator

import (
	"context"
	"fmt"
	"github.com/alchematik/athanor/backend"
	"github.com/alchematik/athanor/build/value"
	"github.com/alchematik/athanor/interpreter"
	"github.com/alchematik/athanor/state"
	plugin "github.com/hashicorp/go-plugin"
	"os/exec"
	"path/filepath"

	// TODO: This has to be not internal anymore?
	backendpb "github.com/alchematik/athanor/internal/gen/go/proto/backend/v1"
	statepb "github.com/alchematik/athanor/internal/gen/go/proto/state/v1"
)

type Evaluator struct {
	ResourceEvaluator ResourceEvaluator
}

type ResourceEvaluator interface {
	EvaluateResource(context.Context, state.Environment, value.Resource) (state.Resource, error)
}

func (e Evaluator) Evaluate(ctx context.Context, env interpreter.Environment) (state.Environment, error) {
	indegrees := map[string]int{}
	for parent, children := range env.DependencyMap {
		if _, ok := indegrees[parent]; !ok {
			indegrees[parent] = 0
		}

		for _, child := range children {
			indegrees[child]++
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
		Objects:       map[string]state.Type{},
		DependencyMap: env.DependencyMap,
	}
	for len(queue) > 0 {
		var alias string
		alias, queue = queue[0], queue[1:]

		fmt.Printf("evaluating: %q\n", alias)

		v := env.Objects[alias]
		r, ok := v.(value.Resource)
		if !ok {
			continue
		}

		s, err := e.ResourceEvaluator.EvaluateResource(ctx, stateEnv, r)
		if err != nil {
			return state.Environment{}, err
		}
		stateEnv.Objects[alias] = s

		for _, childAlias := range env.DependencyMap[alias] {
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
}

type PlanResourceEvaluator struct {
	ValueResolver ValueResolver
}

func (e PlanResourceEvaluator) EvaluateResource(ctx context.Context, env state.Environment, r value.Resource) (state.Resource, error) {
	id, err := e.ValueResolver.ResolveValue(ctx, env, r.Identifier)
	if err != nil {
		return state.Resource{}, err
	}

	config, err := e.ValueResolver.ResolveValue(ctx, env, r.Config)
	if err != nil {
		return state.Resource{}, err
	}

	return state.Resource{
		Identifier: id,
		Config:     config,
		Attrs:      state.Unknown{},
	}, nil
}

type RemoteResourceEvaluator struct {
	ProviderPluginDir string
	ValueResolver     ValueResolver
}

func (e RemoteResourceEvaluator) EvaluateResource(ctx context.Context, env state.Environment, r value.Resource) (state.Resource, error) {
	id, err := e.ValueResolver.ResolveValue(ctx, env, r.Identifier)
	if err != nil {
		return state.Resource{}, err
	}

	pluginPath := filepath.Join(e.ProviderPluginDir, r.Provider.Name, r.Provider.Version, "provider")
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
		Type:       r.ResourceType,
		Identifier: protoID,
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

	return state.Resource{
		Identifier: id,
		Config:     config,
		Attrs:      attrs,
	}, nil
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
	default:
		return nil, fmt.Errorf("unknown type %T\n", val)
	}

}

// TODO: This needs to either
// * Fetch the remote resource using the identifier and fill in the config and attrs fields, or
// * Fill in the config field with the static config and set attrs to something (unresolved?).
// In troduce an interface to evaulate resources?

type RealValueResolver struct {
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
	case value.Unresolved:
		if _, ok := v.Object.(value.Nil); ok {
			obj, inEnv := env.Objects[v.Name]
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
			return state.Unknown{}, nil
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

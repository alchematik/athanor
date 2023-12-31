package reconcile

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/alchematik/athanor/backend"
	"github.com/alchematik/athanor/diff"
	backendpb "github.com/alchematik/athanor/internal/gen/go/proto/backend/v1"
	statepb "github.com/alchematik/athanor/internal/gen/go/proto/state/v1"
	"github.com/alchematik/athanor/state"

	plugin "github.com/hashicorp/go-plugin"
)

type Reconciler struct {
	ProviderPluginDir string
}

func (r Reconciler) ReconcileEnvironment(ctx context.Context, d diff.Environment) (state.Environment, error) {
	indegrees := map[string]int{}
	parentToChildren := map[string][]string{}
	for child, parents := range d.Dependencies {
		indegrees[child] = len(parents)
		for _, parent := range parents {
			parentToChildren[parent] = append(parentToChildren[parent], child)
		}
	}

	var queue []string
	for alias, degrees := range indegrees {
		if degrees == 0 {
			queue = append(queue, alias)
			delete(indegrees, alias)
		}
	}

	reconciledEnv := state.Environment{
		Resources:     map[string]state.Resource{},
		DependencyMap: d.Dependencies,
	}

	// TODO: parallelize.
	for len(queue) > 0 {
		var alias string
		alias, queue = queue[0], queue[1:]

		resourceDiff, ok := d.Diffs[alias].(diff.Resource)
		if !ok {
			return state.Environment{}, fmt.Errorf("expected resource diff, got %T", d.Diffs[alias])
		}

		fmt.Printf("RESOLVING: %v\n", alias)
		resolved, err := resolve(reconciledEnv, resourceDiff.To)
		if err != nil {
			return state.Environment{}, err
		}

		resolvedResource, ok := resolved.(state.Resource)
		if !ok {
			return state.Environment{}, fmt.Errorf("expected resource, got %T", resolved)
		}

		updatedDiff, err := diff.ResourceDiff(resourceDiff.From, resolvedResource)
		if err != nil {
			return state.Environment{}, err
		}

		val, err := r.ReconcileResource(ctx, reconciledEnv, updatedDiff)
		if err != nil {
			return state.Environment{}, err
		}

		reconciledEnv.Resources[alias] = val

		for _, child := range parentToChildren[alias] {
			indegrees[child]--
			if indegrees[child] == 0 {
				queue = append(queue, child)
				delete(indegrees, child)
			}
		}
	}

	return reconciledEnv, nil
}

func (r Reconciler) ReconcileResource(ctx context.Context, env state.Environment, d diff.Resource) (state.Resource, error) {
	switch d.Operation() {
	case diff.OperationNoop:
		return d.To, nil
	case diff.OperationCreate:
		resource := d.To

		pluginPath := filepath.Join(r.ProviderPluginDir, resource.Provider.Name, resource.Provider.Version, "provider")
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

		protoID, err := convertProtoValue(resource.Identifier)
		if err != nil {
			return state.Resource{}, err
		}

		protoConfig, err := convertProtoValue(resource.Config)
		if err != nil {
			return state.Resource{}, err
		}

		// TODO: Check response and make sure we've reconciled.
		res, err := plug.CreateResource(ctx, &backendpb.CreateResourceRequest{
			Identifier: protoID.GetIdentifier(),
			Config:     protoConfig,
		})
		if err != nil {
			return state.Resource{}, err
		}

		resConfig, err := protoToState(res.GetResource().GetConfig())
		if err != nil {
			return state.Resource{}, err
		}

		resAttrs, err := protoToState(res.GetResource().GetAttrs())
		if err != nil {
			return state.Resource{}, err
		}

		r := state.Resource{
			Provider:   resource.Provider,
			Identifier: resource.Identifier,
			Config:     resConfig,
			Attrs:      resAttrs,
		}

		return r, nil
	case diff.OperationDelete:
		resource := d.To

		pluginPath := filepath.Join(r.ProviderPluginDir, resource.Provider.Name, resource.Provider.Version, "provider")
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

		protoID, err := convertProtoValue(resource.Identifier)
		if err != nil {
			return state.Resource{}, err
		}

		// TODO: Check resource state and keep reconciling if we have to.
		_, err = plug.DeleteResource(ctx, &backendpb.DeleteResourceRequest{
			Identifier: protoID.GetIdentifier(),
		})
		if err != nil {
			return state.Resource{}, err
		}

		return resource, nil
	case diff.OperationUpdate:
		resource := d.To

		pluginPath := filepath.Join(r.ProviderPluginDir, resource.Provider.Name, resource.Provider.Version, "provider")
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

		protoID, err := convertProtoValue(resource.Identifier)
		if err != nil {
			return state.Resource{}, err
		}

		protoConfig, err := convertProtoValue(resource.Config)
		if err != nil {
			return state.Resource{}, err
		}

		mask, err := diffToUpdateMask(d.ConfigDiff)
		if err != nil {
			return state.Resource{}, err
		}

		res, err := plug.UpdateResource(ctx, &backendpb.UpdateResourceRequest{
			Identifier: protoID.GetIdentifier(),
			Config:     protoConfig,
			Mask:       mask,
		})
		if err != nil {
			return state.Resource{}, err
		}

		resConfig, err := protoToState(res.GetResource().GetConfig())
		if err != nil {
			return state.Resource{}, err
		}

		resAttrs, err := protoToState(res.GetResource().GetAttrs())
		if err != nil {
			return state.Resource{}, err
		}

		return state.Resource{
			Provider:   resource.Provider,
			Identifier: resource.Identifier,
			Config:     resConfig,
			Attrs:      resAttrs,
		}, nil
	default:
		return state.Resource{}, fmt.Errorf("unsupported operation: %v\n", d.Operation())
	}
}

func diffToUpdateMask(d diff.Type) ([]*backendpb.Field, error) {
	switch t := d.(type) {
	case diff.Resource:
		return diffToUpdateMask(t.ConfigDiff)
	case diff.Map:
		var fields []*backendpb.Field
		for k, v := range t.Diffs {
			// Skip noops and deletes.
			if v.Operation() == diff.OperationNoop || v.Operation() == diff.OperationDelete {
				continue
			}

			sub, err := diffToUpdateMask(v)
			if err != nil {
				return nil, err
			}

			fields = append(fields, &backendpb.Field{Name: k, SubFields: sub})
		}

		return fields, nil
	case diff.String:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported type for mask %T\n", d)
	}
}

// TODO: Exctact into common package.
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
		return nil, fmt.Errorf("unsupported type for converting %v", val)
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
		return nil, fmt.Errorf("reconciler: unknown type %T\n", val)
	}
}

func resolve(env state.Environment, res state.Type) (state.Type, error) {
	fmt.Printf("resolve: %T, %+v\n", res, res)
	switch r := res.(type) {
	case state.String:
		return r, nil
	case state.Map:
		m := state.Map{
			Entries: map[string]state.Type{},
		}
		for k, v := range r.Entries {
			resolved, err := resolve(env, v)
			if err != nil {
				return nil, err
			}

			m.Entries[k] = resolved
		}

		return m, nil
	case state.ResourceRef:
		res, ok := env.Resources[r.Alias]
		if !ok {
			return nil, fmt.Errorf("resolve: no resource with alias %q found", r.Alias)
		}

		return res, nil
	case state.Resource:
		config, err := resolve(env, r.Config)
		if err != nil {
			return nil, err
		}

		return state.Resource{
			Provider:   r.Provider,
			Identifier: r.Identifier,
			Config:     config,
			Attrs:      r.Attrs,
		}, nil
	case state.Unknown:
		resolved, err := resolve(env, r.Object)
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
		case state.Map:
			m = obj.Entries
		default:
			return nil, fmt.Errorf("value type %T has no field %q", obj, r.Name)
		}

		val, ok := m[r.Name]
		if !ok {
			return nil, fmt.Errorf("property %q not set", r.Name)
		}

		return val, nil
	default:
		return nil, fmt.Errorf("invalid type to resolve: %T", res)
	}
}

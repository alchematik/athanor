package resource

import (
	"context"
	"fmt"

	backendpb "github.com/alchematik/athanor/internal/gen/go/proto/backend/v1"
	statepb "github.com/alchematik/athanor/internal/gen/go/proto/state/v1"
	"github.com/alchematik/athanor/plugin"
	"github.com/alchematik/athanor/state"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Field struct {
	Name      string
	SubFields []Field
}

type API struct {
	ProviderPluginManager plugin.Provider
}

func (a API) GetResource(ctx context.Context, r state.Resource) (state.Resource, error) {
	client, err := a.ProviderPluginManager.Client(r.Provider)
	if err != nil {
		return state.Resource{}, err
	}

	id, err := toProto(r.Identifier)
	if err != nil {
		return state.Resource{}, err
	}

	request := &backendpb.GetResourceRequest{
		Identifier: id.GetIdentifier(),
	}
	response, err := client.GetResource(ctx, request)
	exists := state.Bool{Value: true}
	if err != nil {
		if status.Code(err) == codes.NotFound {
			exists.Value = false
		} else {
			return state.Resource{}, err
		}
	}

	config, err := fromProto(response.GetResource().GetConfig())
	if err != nil {
		return state.Resource{}, err
	}

	attrs, err := fromProto(response.GetResource().GetAttrs())
	if err != nil {
		return state.Resource{}, err
	}

	return state.Resource{
		Provider:   r.Provider,
		Identifier: r.Identifier,
		Config:     config,
		Attrs:      attrs,
		Exists:     exists,
	}, nil
}

func (a API) CreateResource(ctx context.Context, r state.Resource) (state.Resource, error) {
	client, err := a.ProviderPluginManager.Client(r.Provider)
	if err != nil {
		return state.Resource{}, err
	}

	id, err := toProto(r.Identifier)
	if err != nil {
		return state.Resource{}, err
	}

	config, err := toProto(r.Config)
	if err != nil {
		return state.Resource{}, err
	}

	request := &backendpb.CreateResourceRequest{
		Identifier: id.GetIdentifier(),
		Config:     config,
	}
	response, err := client.CreateResource(ctx, request)
	if err != nil {
		return state.Resource{}, err
	}

	resConfig, err := fromProto(response.GetResource().GetConfig())
	if err != nil {
		return state.Resource{}, err
	}

	attrs, err := fromProto(response.GetResource().GetAttrs())
	if err != nil {
		return state.Resource{}, err
	}

	return state.Resource{
		Provider:   r.Provider,
		Identifier: r.Identifier,
		Config:     resConfig,
		Attrs:      attrs,
		Exists:     state.Bool{Value: true},
	}, nil
}

func (a API) DeleteResource(ctx context.Context, r state.Resource) error {
	client, err := a.ProviderPluginManager.Client(r.Provider)
	if err != nil {
		return err
	}

	id, err := toProto(r.Identifier)
	if err != nil {
		return err
	}

	request := &backendpb.DeleteResourceRequest{
		Identifier: id.GetIdentifier(),
	}
	_, err = client.DeleteResource(ctx, request)
	if err != nil {
		return err
	}

	return nil
}

func (a API) UpdateResource(ctx context.Context, r state.Resource, mask []Field) (state.Resource, error) {
	client, err := a.ProviderPluginManager.Client(r.Provider)
	if err != nil {
		return state.Resource{}, err
	}

	id, err := toProto(r.Identifier)
	if err != nil {
		return state.Resource{}, err
	}

	config, err := toProto(r.Config)
	if err != nil {
		return state.Resource{}, err
	}

	request := &backendpb.UpdateResourceRequest{
		Identifier: id.GetIdentifier(),
		Config:     config,
		Mask:       toProtoMask(mask),
	}
	response, err := client.UpdateResource(ctx, request)
	if err != nil {
		return state.Resource{}, err
	}

	responseConfig, err := fromProto(response.GetResource().GetConfig())
	if err != nil {
		return state.Resource{}, err
	}

	responseAttrs, err := fromProto(response.GetResource().GetAttrs())
	if err != nil {
		return state.Resource{}, err
	}

	return state.Resource{
		Identifier: r.Identifier,
		Config:     responseConfig,
		Attrs:      responseAttrs,
		Exists:     state.Bool{Value: true},
	}, nil
}

func toProtoMask(mask []Field) []*backendpb.Field {
	var protoMask []*backendpb.Field
	for _, f := range mask {
		p := &backendpb.Field{
			Name:      f.Name,
			SubFields: toProtoMask(f.SubFields),
		}
		protoMask = append(protoMask, p)
	}

	return protoMask
}

func fromProto(val *statepb.Value) (state.Type, error) {
	switch v := val.GetType().(type) {
	case *statepb.Value_Map:
		entries := map[string]state.Type{}
		for k, element := range v.Map.GetEntries() {
			converted, err := fromProto(element)
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

func toProto(val state.Type) (*statepb.Value, error) {
	switch v := val.(type) {
	case state.String:
		return &statepb.Value{
			Type: &statepb.Value_StringValue{StringValue: v.Value},
		}, nil
	case state.Map:
		entries := map[string]*statepb.Value{}
		for k, v := range v.Entries {
			converted, err := toProto(v)
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
		converted, err := toProto(v.Value)
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

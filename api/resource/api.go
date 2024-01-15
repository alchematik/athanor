package resource

import (
	"context"
	"fmt"

	providerpb "github.com/alchematik/athanor/internal/gen/go/proto/provider/v1"
	"github.com/alchematik/athanor/plugin"
	"github.com/alchematik/athanor/state"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Field struct {
	Name      string
	Operation Operation
	SubFields []Field
}

type Operation string

const (
	OperationUpdate Operation = "update"
	OperationDelete Operation = "delete"
)

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

	request := &providerpb.GetResourceRequest{
		Identifier: id,
	}
	response, err := client.GetResource(ctx, request)
	exists := state.Bool{Value: true}
	if err != nil {
		if status.Code(err) == codes.NotFound {
			exists.Value = false
		} else {
			return state.Resource{}, fmt.Errorf("get resource: %v", err)
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

	request := &providerpb.CreateResourceRequest{
		Identifier: id,
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

	request := &providerpb.DeleteResourceRequest{
		Identifier: id,
	}
	_, err = client.DeleteResource(ctx, request)
	return err
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

	request := &providerpb.UpdateResourceRequest{
		Identifier: id,
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
		Provider:   r.Provider,
		Identifier: r.Identifier,
		Config:     responseConfig,
		Attrs:      responseAttrs,
		Exists:     state.Bool{Value: true},
	}, nil
}

func toProtoMask(mask []Field) []*providerpb.Field {
	var protoMask []*providerpb.Field
	for _, f := range mask {
		op := providerpb.Operation_OPERATION_UPDATE
		if f.Operation == OperationDelete {
			op = providerpb.Operation_OPERATION_DELETE
		}
		p := &providerpb.Field{
			Name:      f.Name,
			SubFields: toProtoMask(f.SubFields),
			Operation: op,
		}
		protoMask = append(protoMask, p)
	}

	return protoMask
}

func fromProto(val *providerpb.Value) (state.Type, error) {
	switch v := val.GetType().(type) {
	case *providerpb.Value_Map:
		entries := map[string]state.Type{}
		for k, element := range v.Map.GetEntries() {
			converted, err := fromProto(element)
			if err != nil {
				return nil, err
			}
			entries[k] = converted
		}

		return state.Map{Entries: entries}, nil
	case *providerpb.Value_StringValue:
		return state.String{Value: v.StringValue}, nil
	case nil:
		return state.Nil{}, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", val.GetType())
	}
}

func toProto(val state.Type) (*providerpb.Value, error) {
	switch v := val.(type) {
	case state.String:
		return &providerpb.Value{
			Type: &providerpb.Value_StringValue{StringValue: v.Value},
		}, nil
	case state.Map:
		entries := map[string]*providerpb.Value{}
		for k, v := range v.Entries {
			converted, err := toProto(v)
			if err != nil {
				return nil, err
			}
			entries[k] = converted
		}

		return &providerpb.Value{
			Type: &providerpb.Value_Map{
				Map: &providerpb.MapValue{
					Entries: entries,
				},
			},
		}, nil
	case state.Identifier:
		converted, err := toProto(v.Value)
		if err != nil {
			return nil, err
		}

		return &providerpb.Value{
			Type: &providerpb.Value_Identifier{
				Identifier: &providerpb.Identifier{
					Type:  v.ResourceType,
					Value: converted,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("convert proto: unknown type %T\n", val)
	}

}

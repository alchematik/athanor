package main

import (
	"context"
	"os/exec"

	"github.com/alchematik/athanor/backend"
	backendpb "github.com/alchematik/athanor/internal/gen/go/proto/backend/v1"
	translatorpb "github.com/alchematik/athanor/internal/gen/go/proto/translator/v1"
	"github.com/alchematik/athanor/translator"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type Server struct {
}

func (s *Server) GetResource(ctx context.Context, req *backendpb.GetResourceRequest) (*backendpb.GetResourceResponse, error) {
	pluginPath := "./.translators/json/v0.0.1/translator"
	handle := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: translator.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"translator": &translator.Plugin{},
		},
		Cmd:              exec.Command("sh", "-c", pluginPath),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
	})

	dispensor, err := handle.Client()
	if err != nil {
		return &backendpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	raw, err := dispensor.Dispense("translator")
	if err != nil {
		return &backendpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	translatorClient := raw.(translatorpb.TranslatorClient)

	path := req.GetIdentifier()[0].GetValue().GetStringValue()

	out, err := translatorClient.ReadConsumerBlueprint(ctx, &translatorpb.ReadConsumerBlueprintRequest{
		Path: path,
	})
	if err != nil {
		return &backendpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	var fields []*backendpb.Field

	var config []*structpb.Value
	for _, consumerResource := range out.Blueprint.Resources {
		// TODO: resolve field loaders and serialize values.

		r := &backendpb.Resource{
			Type: consumerResource.GetType(),
		}

		val, err := resourceToValue(r)
		if err != nil {
			return &backendpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
		}

		config = append(config, val)
	}

	fields = append(fields, &backendpb.Field{
		Name: "config",
		Type: "fields",
		Value: structpb.NewListValue(&structpb.ListValue{
			Values: config,
		}),
	})

	var identifierFields []*structpb.Value
	for _, f := range req.GetIdentifier() {
		val, err := fieldToValue(f)
		if err != nil {
			return &backendpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
		}
		identifierFields = append(identifierFields, val)
	}

	fields = append(fields, &backendpb.Field{
		Name: "identifier",
		Type: "fields",
		Value: structpb.NewListValue(&structpb.ListValue{
			Values: identifierFields,
		}),
	})

	return &backendpb.GetResourceResponse{
		Resource: &backendpb.Resource{
			Provider: &backendpb.Provider{
				Name: "athanor",
			},
			Fields: fields,
		},
	}, nil
}

func resourceToValue(r *backendpb.Resource) (*structpb.Value, error) {
	m := map[string]any{}
	m["type"] = r.GetType()

	return structpb.NewValue(m)
}

func fieldToValue(f *backendpb.Field) (*structpb.Value, error) {
	m := map[string]any{}
	m["name"] = f.Name
	m["type"] = f.Type
	// m["value"] = f.Value

	// switch f.Type {
	// case "string":
	// 	m["value"] = f.Value.GetStringValue()
	// }

	return structpb.NewValue(m)
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: backend.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"backend": &backend.Plugin{
				BackendServer: &Server{},
			},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

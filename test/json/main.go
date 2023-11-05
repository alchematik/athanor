package main

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/go-plugin"
	"os"

	blueprintpb "github.com/alchematik/athanor/internal/gen/go/proto/blueprint/v1"
	translatorpb "github.com/alchematik/athanor/internal/gen/go/proto/translator/v1"
	"github.com/alchematik/athanor/translator"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
}

func (s *Server) ReadProviderBlueprint(ctx context.Context, req *translatorpb.ReadProviderBlueprintRequest) (*translatorpb.ReadProviderBluepintResponse, error) {
	path := req.GetPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return &translatorpb.ReadProviderBluepintResponse{}, status.Error(codes.Internal, err.Error())
	}

	var bp *blueprintpb.ProviderBlueprint
	if err := json.Unmarshal(data, &bp); err != nil {
		return &translatorpb.ReadProviderBluepintResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &translatorpb.ReadProviderBluepintResponse{
		Blueprint: bp,
	}, nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: translator.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"translator": &translator.Plugin{
				TranslatorServer: &Server{},
			},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

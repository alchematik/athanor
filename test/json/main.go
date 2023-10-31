package main

import (
	"context"
	"github.com/hashicorp/go-plugin"

	"github.com/alchematik/athanor/translator"

	translatorpb "github.com/alchematik/athanor/internal/gen/go/proto/translator/v1"
)

type Server struct {
}

func (s *Server) ReadProviderBlueprint(ctx context.Context, req *translatorpb.ReadProviderBlueprintRequest) (*translatorpb.ReadProviderBluepintResponse, error) {
	return &translatorpb.ReadProviderBluepintResponse{}, nil
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

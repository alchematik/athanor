package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	consumerpb "github.com/alchematik/athanor/internal/gen/go/proto/consumer/v1"
	providerpb "github.com/alchematik/athanor/internal/gen/go/proto/provider/v1"
	translatorpb "github.com/alchematik/athanor/internal/gen/go/proto/translator/v1"
	"github.com/alchematik/athanor/translator"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
}

func (s *Server) ReadProviderBlueprint(ctx context.Context, req *translatorpb.ReadProviderBlueprintRequest) (*translatorpb.ReadProviderBlueprintResponse, error) {
	path := req.GetPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return &translatorpb.ReadProviderBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}

	var bp *providerpb.Blueprint
	if err := json.Unmarshal(data, &bp); err != nil {
		return &translatorpb.ReadProviderBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &translatorpb.ReadProviderBlueprintResponse{
		Blueprint: bp,
	}, nil
}

func (s *Server) ReadConsumerBlueprint(ctx context.Context, req *translatorpb.ReadConsumerBlueprintRequest) (*translatorpb.ReadConsumerBlueprintResponse, error) {
	path := req.GetPath()
	var resources []*consumerpb.BlueprintResource
	err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if err != nil {
			return err
		}

		ext := filepath.Ext(path)
		if ext != ".json" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var bp consumerpb.Blueprint
		if err := json.Unmarshal(data, &bp); err != nil {
			return fmt.Errorf("error unmarshaling json: %v\n", err)
		}

		resources = append(resources, bp.GetResources()...)

		return nil
	})
	if err != nil {
		return &translatorpb.ReadConsumerBlueprintResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &translatorpb.ReadConsumerBlueprintResponse{
		Blueprint: &consumerpb.Blueprint{},
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

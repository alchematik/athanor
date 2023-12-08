package main

import (
	"context"

	"github.com/alchematik/athanor/backend"
	backendpb "github.com/alchematik/athanor/internal/gen/go/proto/backend/v1"
	statepb "github.com/alchematik/athanor/internal/gen/go/proto/state/v1"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
}

func (s *Server) GetResource(ctx context.Context, req *backendpb.GetResourceRequest) (*backendpb.GetResourceResponse, error) {
	t := req.GetType()
	switch t {
	case "bucket":
		r := &statepb.Resource{
			Identifier: req.GetIdentifier(),
			Config: &statepb.Value{
				Type: &statepb.Value_Map{
					Map: &statepb.MapValue{
						Entries: map[string]*statepb.Value{
							"expiration": &statepb.Value{Type: &statepb.Value_StringValue{StringValue: "1d"}},
						},
					},
				},
			},
			Attrs: &statepb.Value{
				Type: &statepb.Value_Map{
					Map: &statepb.MapValue{
						Entries: map[string]*statepb.Value{
							"bar": &statepb.Value{
								Type: &statepb.Value_Map{
									Map: &statepb.MapValue{
										Entries: map[string]*statepb.Value{
											"foo": &statepb.Value{Type: &statepb.Value_StringValue{StringValue: "hi"}},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		return &backendpb.GetResourceResponse{
			Resource: r,
		}, nil
	case "bucket_object":
		r := &statepb.Resource{
			Identifier: req.GetIdentifier(),
			Config: &statepb.Value{
				Type: &statepb.Value_Map{
					Map: &statepb.MapValue{
						Entries: map[string]*statepb.Value{
							"contents": &statepb.Value{Type: &statepb.Value_StringValue{StringValue: "blablablablabla"}},
						},
					},
				},
			},
			Attrs: &statepb.Value{
				Type: &statepb.Value_Map{
					Map: &statepb.MapValue{
						Entries: map[string]*statepb.Value{},
					},
				},
			},
		}
		return &backendpb.GetResourceResponse{
			Resource: r,
		}, nil
	default:
		return &backendpb.GetResourceResponse{}, status.Errorf(codes.InvalidArgument, "unsupported type %q", t)
	}
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

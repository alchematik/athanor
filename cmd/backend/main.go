package main

import (
	"context"

	"github.com/alchematik/athanor/backend"
	backendpb "github.com/alchematik/athanor/internal/gen/go/proto/provider/v1"
	statepb "github.com/alchematik/athanor/internal/gen/go/proto/state/v1"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
}

func (s *Server) GetResource(ctx context.Context, req *backendpb.GetResourceRequest) (*backendpb.GetResourceResponse, error) {
	t := req.GetIdentifier().GetType()
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
							"contents":   {Type: &statepb.Value_StringValue{StringValue: "blablablablabla"}},
							"some_field": {Type: &statepb.Value_StringValue{StringValue: "hehe"}},
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

func (s *Server) CreateResource(ctx context.Context, req *backendpb.CreateResourceRequest) (*backendpb.CreateResourceResponse, error) {
	var r *statepb.Resource
	switch req.GetIdentifier().GetType() {
	case "bucket":
		r = &statepb.Resource{
			Identifier: req.GetIdentifier(),
			Config: &statepb.Value{
				Type: &statepb.Value_Map{
					Map: &statepb.MapValue{
						Entries: map[string]*statepb.Value{
							"expiration": &statepb.Value{Type: &statepb.Value_StringValue{StringValue: "12h"}},
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
	case "bucket_object":
		r = &statepb.Resource{
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
	}
	return &backendpb.CreateResourceResponse{
		Resource: r,
	}, nil
}

func (s *Server) UpdateResource(ctx context.Context, req *backendpb.UpdateResourceRequest) (*backendpb.UpdateResourceResponse, error) {
	var r *statepb.Resource
	switch req.GetIdentifier().GetType() {
	case "bucket":
		r = &statepb.Resource{
			Identifier: req.GetIdentifier(),
			Config: &statepb.Value{
				Type: &statepb.Value_Map{
					Map: &statepb.MapValue{
						Entries: map[string]*statepb.Value{
							"expiration": &statepb.Value{Type: &statepb.Value_StringValue{StringValue: "12h"}},
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
	case "bucket_object":
		r = &statepb.Resource{
			Identifier: req.GetIdentifier(),
			Config: &statepb.Value{
				Type: &statepb.Value_Map{
					Map: &statepb.MapValue{
						Entries: map[string]*statepb.Value{
							"contents":   &statepb.Value{Type: &statepb.Value_StringValue{StringValue: "blablablablabla"}},
							"some_field": &statepb.Value{Type: &statepb.Value_StringValue{StringValue: "hi"}},
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
	default:
		return &backendpb.UpdateResourceResponse{}, status.Errorf(codes.InvalidArgument, "requires resource type")
	}
	return &backendpb.UpdateResourceResponse{
		Resource: r,
	}, nil
}

func (s *Server) DeleteResource(ctx context.Context, req *backendpb.DeleteResourceRequest) (*backendpb.DeleteResourceResponse, error) {
	return &backendpb.DeleteResourceResponse{}, nil
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

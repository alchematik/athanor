package main

import (
	"context"
	"fmt"

	backendpb "github.com/alchematik/athanor/internal/gen/go/proto/provider/v1"
	statepb "github.com/alchematik/athanor/internal/gen/go/proto/provider/v1"
	sdk "github.com/alchematik/athanor/sdk/go/provider/value"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	ResourceHandlers map[string]ResourceHandler
}

func ParseBucketIdentifier(val *statepb.Identifier) BucketIdentifier {
	account := val.GetValue().GetMap().GetEntries()["account"].GetStringValue()
	region := val.GetValue().GetMap().GetEntries()["region"].GetStringValue()
	name := val.GetValue().GetMap().GetEntries()["name"].GetStringValue()

	return BucketIdentifier{
		Account: sdk.String(account),
		Region:  sdk.String(region),
		Name:    sdk.String(name),
	}
}

type BucketIdentifier struct {
	Account sdk.StringType
	Region  sdk.StringType
	Name    sdk.StringType
}

func (id BucketIdentifier) ResourceType() string {
	return "bucket"
}

func (id BucketIdentifier) Value() sdk.Type {
	return sdk.Map(map[string]sdk.Type{
		"account": id.Account,
		"region":  id.Region,
		"name":    id.Name,
	})
}

func (id BucketIdentifier) ToStateValue() sdk.StateValue {
	return sdk.IdentifierStateValue{
		ResourceType: id.ResourceType(),
		Value:        id.Value().ToStateValue(),
	}
}

func ParseBucketConfig(val *statepb.Value) BucketConfig {
	expiration := val.GetMap().GetEntries()["expiration"].GetStringValue()
	return BucketConfig{
		Expiration: sdk.String(expiration),
	}
}

type BucketConfig struct {
	Expiration sdk.StringType
}

func (c BucketConfig) ToStateValue() sdk.StateValue {
	return sdk.Map(map[string]sdk.Type{
		"expiration": c.Expiration,
	}).ToStateValue()
}

type BucketAttrs struct {
	Bar Bar
}

func (c BucketAttrs) ToStateValue() sdk.StateValue {
	return sdk.Map(map[string]sdk.Type{
		"bar": c.Bar,
	}).ToStateValue()
}

type Bar struct {
	Foo sdk.StringType
}

func (b Bar) ToStateValue() sdk.StateValue {
	return sdk.Map(map[string]sdk.Type{
		"foo": b.Foo,
	}).ToStateValue()
}

type BucketObjectIdentifier struct {
	Bucket sdk.IdentifierType
	Name   sdk.StringType
}

func (id BucketObjectIdentifier) ResourceType() string {
	return "bucket_object"
}

func (id BucketObjectIdentifier) Value() sdk.Type {
	return sdk.Map(map[string]sdk.Type{
		"bucket": id.Bucket,
		"name":   id.Name,
	})
}

func (id BucketObjectIdentifier) ToStateValue() sdk.StateValue {
	return sdk.IdentifierStateValue{
		ResourceType: id.ResourceType(),
		Value:        id.Value().ToStateValue(),
	}
}

func ParseBucketObjectConfig(val *statepb.Value) BucketObjectConfig {
	return BucketObjectConfig{
		Contents:  sdk.String(val.GetMap().GetEntries()["contents"].GetStringValue()),
		SomeField: sdk.String(val.GetMap().GetEntries()["some_field"].GetStringValue()),
	}
}

type BucketObjectConfig struct {
	Contents  sdk.StringType
	SomeField sdk.StringType
}

func (b BucketObjectConfig) ToStateValue() sdk.StateValue {
	return sdk.Map(map[string]sdk.Type{
		"contents":   b.Contents,
		"some_field": b.SomeField,
	}).ToStateValue()
}

type BucketObjectAttrs struct {
}

func (b BucketObjectAttrs) ToStateValue() sdk.StateValue {
	return sdk.Map(map[string]sdk.Type{}).ToStateValue()
}

func ParseBucketObjectIdentifier(id *statepb.Identifier) BucketObjectIdentifier {
	bucket := ParseBucketIdentifier(id.GetValue().GetMap().GetEntries()["bucket"].GetIdentifier())

	name := id.GetValue().GetMap().GetEntries()["name"].GetStringValue()

	return BucketObjectIdentifier{
		Bucket: bucket,
		Name:   sdk.String(name),
	}
}

type BucketGetter interface {
	GetBucket(context.Context, BucketIdentifier) (sdk.Resource, error)
}

type BucketCreator interface {
	CreateBucket(context.Context, BucketIdentifier, BucketConfig) (sdk.Resource, error)
}

type BucketUpdator interface {
	UpdateBucket(context.Context, BucketIdentifier, BucketConfig) (sdk.Resource, error)
}

type BucketDeleter interface {
	DeleteBucket(context.Context, BucketIdentifier) error
}

type BucketObjectGetter interface {
	GetBucketObject(context.Context, BucketObjectIdentifier) (sdk.Resource, error)
}

type BucketObjectCreator interface {
	CreateBucketObject(context.Context, BucketObjectIdentifier, BucketObjectConfig) (sdk.Resource, error)
}

type BucketObjectUpdator interface {
	UpdateBucketObject(context.Context, BucketObjectIdentifier, BucketObjectConfig) (sdk.Resource, error)
}

type BucketObjectDeleter interface {
	DeleteBucketObject(context.Context, BucketIdentifier) error
}

type ResourceHandler interface {
	GetResource(context.Context, *statepb.Identifier) (*statepb.Resource, error)
	CreateResource(context.Context, *statepb.Identifier, *statepb.Value) (*statepb.Resource, error)
}

type BucketHandler struct {
	BucketGetter  BucketGetter
	BucketCreator BucketCreator
}

type BucketObjectHandler struct {
	BucketObjectGetter  BucketObjectGetter
	BucketObjectCreator BucketObjectCreator
}

func (h BucketHandler) GetResource(ctx context.Context, id *statepb.Identifier) (*statepb.Resource, error) {
	if h.BucketGetter == nil {
		return nil, fmt.Errorf("unimplemented")
	}

	bucketID := ParseBucketIdentifier(id)
	res, err := h.BucketGetter.GetBucket(ctx, bucketID)
	if err != nil {
		return nil, err
	}

	return res.ToStateValue().ToStateValueProto(), nil
}

func (h BucketHandler) CreateResource(ctx context.Context, id *statepb.Identifier, config *statepb.Value) (*statepb.Resource, error) {
	if h.BucketCreator == nil {
		return nil, fmt.Errorf("unimplemented")
	}

	bucketID := ParseBucketIdentifier(id)
	bucketConfig := ParseBucketConfig(config)

	res, err := h.BucketCreator.CreateBucket(ctx, bucketID, bucketConfig)
	if err != nil {
		return nil, err
	}

	return res.ToStateValue().ToStateValueProto(), nil
}

func (h BucketObjectHandler) GetResource(ctx context.Context, id *statepb.Identifier) (*statepb.Resource, error) {
	if h.BucketObjectGetter == nil {
		return nil, fmt.Errorf("unimplemented")
	}

	bucketObjectID := ParseBucketObjectIdentifier(id)
	res, err := h.BucketObjectGetter.GetBucketObject(ctx, bucketObjectID)
	if err != nil {
		return nil, err
	}

	return res.ToStateValue().ToStateValueProto(), nil
}

func (h BucketObjectHandler) CreateResource(ctx context.Context, id *statepb.Identifier, config *statepb.Value) (*statepb.Resource, error) {
	if h.BucketObjectCreator == nil {
		return nil, fmt.Errorf("unimplemented")
	}

	bucketObjectID := ParseBucketObjectIdentifier(id)
	bucketObjectConfig := ParseBucketObjectConfig(config)

	res, err := h.BucketObjectCreator.CreateBucketObject(ctx, bucketObjectID, bucketObjectConfig)
	if err != nil {
		return nil, err
	}

	return res.ToStateValue().ToStateValueProto(), nil
}

type Bucket struct {
}

func (b Bucket) GetBucket(ctx context.Context, id BucketIdentifier) (sdk.Resource, error) {
	config := BucketConfig{
		Expiration: "1d",
	}
	attrs := BucketAttrs{
		Bar: Bar{
			Foo: "hi",
		},
	}

	return sdk.Resource{
		Identifier: id,
		Config:     config,
		Attrs:      attrs,
	}, nil
}

func (b Bucket) CreateBucket(ctx context.Context, id BucketIdentifier, config BucketConfig) (sdk.Resource, error) {
	config.Expiration = "12h"

	attrs := BucketAttrs{
		Bar: Bar{
			Foo: "hi",
		},
	}

	return sdk.Resource{
		Identifier: id,
		Config:     config,
		Attrs:      attrs,
	}, nil
}

type BucketObject struct{}

func (b BucketObject) GetBucketObject(ctx context.Context, id BucketObjectIdentifier) (sdk.Resource, error) {
	config := BucketObjectConfig{
		Contents:  "blablabla",
		SomeField: "hehe",
	}

	attrs := BucketObjectAttrs{}

	return sdk.Resource{
		Identifier: id,
		Config:     config,
		Attrs:      attrs,
	}, nil
}

func (b BucketObject) CreateBucketObject(ctx context.Context, id BucketObjectIdentifier, config BucketObjectConfig) (sdk.Resource, error) {
	return sdk.Resource{
		Identifier: id,
		Config:     config,
		Attrs:      BucketAttrs{},
	}, nil
}

func (s *Server) GetResource(ctx context.Context, req *backendpb.GetResourceRequest) (*backendpb.GetResourceResponse, error) {
	t := req.GetIdentifier().GetType()
	handler, ok := s.ResourceHandlers[t]
	if !ok {
		return &backendpb.GetResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	res, err := handler.GetResource(ctx, req.GetIdentifier())
	if err != nil {
		return &backendpb.GetResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &backendpb.GetResourceResponse{Resource: res}, nil
}

func (s *Server) CreateResource(ctx context.Context, req *backendpb.CreateResourceRequest) (*backendpb.CreateResourceResponse, error) {
	t := req.GetIdentifier().GetType()
	handler, ok := s.ResourceHandlers[t]
	if !ok {
		return &backendpb.CreateResourceResponse{}, status.Error(codes.NotFound, "resource type not found")
	}

	res, err := handler.CreateResource(ctx, req.GetIdentifier(), req.GetConfig())
	if err != nil {
		return &backendpb.CreateResourceResponse{}, status.Error(codes.Internal, err.Error())
	}

	return &backendpb.CreateResourceResponse{Resource: res}, nil
}

func (s *Server) UpdateResource(ctx context.Context, req *backendpb.UpdateResourceRequest) (*backendpb.UpdateResourceResponse, error) {
	var r *statepb.Resource
	switch req.GetIdentifier().GetType() {
	case "bucket":
		id := ParseBucketIdentifier(req.GetIdentifier())

		config := ParseBucketConfig(req.GetConfig())

		config.Expiration = "12h"

		attrs := BucketAttrs{
			Bar: Bar{
				Foo: "hi",
			},
		}

		r = sdk.Resource{
			Identifier: id,
			Config:     config,
			Attrs:      attrs,
		}.ToStateValue().ToStateValueProto()
	case "bucket_object":
		id := ParseBucketObjectIdentifier(req.GetIdentifier())

		config := ParseBucketObjectConfig(req.GetConfig())
		config.Contents = "blablablablablablabla"
		config.SomeField = "hi"

		attrs := BucketAttrs{}

		r = sdk.Resource{
			Identifier: id,
			Config:     config,
			Attrs:      attrs,
		}.ToStateValue().ToStateValueProto()
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
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "COOKIE",
			MagicCookieValue: "hi",
		},
		Plugins: map[string]plugin.Plugin{
			"backend": &Plugin{
				BackendServer: &Server{
					ResourceHandlers: map[string]ResourceHandler{
						"bucket": BucketHandler{
							BucketGetter:  Bucket{},
							BucketCreator: Bucket{},
						},
						"bucket_object": BucketObjectHandler{
							BucketObjectGetter:  BucketObject{},
							BucketObjectCreator: BucketObject{},
						},
					},
				},
			},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

type Plugin struct {
	plugin.Plugin

	BackendServer backendpb.ProviderServer
}

func (p *Plugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	backendpb.RegisterProviderServer(s, p.BackendServer)
	return nil
}

func (p *Plugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return backendpb.NewProviderClient(conn), nil
}

package main

import (
	"context"

	"github.com/hashicorp/go-plugin"

	"github.com/alchematik/athanor/provider"

	gen "github.com/alchematik/athanor/gen/gcp/v0.0.1"
	bucket "github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"
	bucketobject "github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket_object"
	resourcepolicy "github.com/alchematik/athanor/gen/gcp/v0.0.1/resource_policy"
)

func main() {
	r := gen.ClientRegistry{
		BucketClient:         &bucketClient{},
		BucketObjectClient:   &bucketClient{},
		ResourcePolicyClient: &iamClient{},
	}
	pluginMap := map[string]plugin.Plugin{
		"provider": &provider.ProviderPlugin{
			ClientRegistry: &r,
			Schema:         gen.Schema(),
		},
	}

	handshakeConfig := plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "BASIC_PLUGIN",
		MagicCookieValue: "hello",
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}

type bucketClient struct {
}

func (c *bucketClient) GetBucket(ctx context.Context, id *bucket.Identifier) (*bucket.Bucket, error) {
	return nil, nil
}

func (c *bucketClient) CreateBucket(ctx context.Context, id *bucket.Identifier, config *bucket.Config) error {
	return nil
}

func (c *bucketClient) GetBucketObject(ctx context.Context, id *bucketobject.Identifier) (*bucketobject.BucketObject, error) {
	return nil, nil
}

func (c *bucketClient) CreateBucketObject(ctx context.Context, id *bucketobject.Identifier, config *bucketobject.Config) error {
	return nil
}

type iamClient struct {
}

func (c *iamClient) GetResourcePolicy(ctx context.Context, id *resourcepolicy.Identifier) (*resourcepolicy.ResourcePolicy, error) {
	return nil, nil
}

func (c *iamClient) CreateResourcePolicy(ctx context.Context, id *resourcepolicy.Identifier, config *resourcepolicy.Config) error {
	return nil
}

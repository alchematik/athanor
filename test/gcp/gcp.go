package main

import (
	"context"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	"github.com/alchematik/athanor/provider"

	gen "github.com/alchematik/athanor/gen/gcp/v0.0.1"
	bucket "github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"
	bucketobject "github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket_object"
	resourcepolicy "github.com/alchematik/athanor/gen/gcp/v0.0.1/resource_policy"
	"github.com/alchematik/athanor/gen/gcp/v0.0.1/schema"
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
			SchemaProvider: schemaProvider{},
		},
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Output: os.Stderr,
	})
	logger.Info("STARTING PLUGIN")

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

type schemaProvider struct{}

func (s schemaProvider) ResourceNames() []string {
	p := gen.Parser{}
	return p.ResourceNames()
}

func (s schemaProvider) IdentifierSchema() map[string][]provider.Field {
	return schema.Schema
}

func (s schemaProvider) ConfigSchema() map[string][]provider.Field {
	return schema.ConfigSchema
}

func Parser() any {
	return &gen.Parser{}
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

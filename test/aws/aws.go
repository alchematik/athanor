package main

import (
	"context"

	"github.com/hashicorp/go-plugin"

	"github.com/alchematik/athanor/provider"

	gen "github.com/alchematik/athanor/gen/aws/v0.0.1"
	bucket "github.com/alchematik/athanor/gen/aws/v0.0.1/bucket"
)

func main() {
	r := gen.ClientRegistry{
		BucketClient: &s3Client{},
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

type s3Client struct {
}

func (c *s3Client) GetBucket(ctx context.Context, id *bucket.Identifier) (*bucket.Bucket, error) {
	return nil, nil
}

func (c *s3Client) CreateBucket(ctx context.Context, id *bucket.Identifier, config *bucket.Config) error {
	return nil
}

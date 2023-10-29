package main

import (
	"context"
	"errors"
	// "fmt"
	"net/http"
	"os"

	"cloud.google.com/go/storage"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/api/googleapi"

	"github.com/alchematik/athanor/provider"

	gen "github.com/alchematik/athanor/gen/gcp/v0.0.1"
	bucket "github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"
	bucketobject "github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket_object"
	resourcepolicy "github.com/alchematik/athanor/gen/gcp/v0.0.1/resource_policy"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Output: os.Stderr,
		Level:  hclog.Trace,
	})
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		logger.Error("ERROR >>> ", err)
		return
	}
	r := gen.ClientRegistry{
		BucketClient: &bucketClient{
			client: client,
			logger: logger,
		},
		BucketObjectClient: &bucketClient{
			client: client,
		},
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

	logger.Info("DOING THE THING")

	// logger := hclog.New(&hclog.LoggerOptions{
	// 	Output: os.Stderr,
	// 	Level:  hclog.Debug,
	// })
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
		// Logger:          hclog.NewNullLogger(),
		// Logger: logger,
	})
}

type bucketClient struct {
	client *storage.Client
	logger hclog.Logger
}

func (c *bucketClient) GetBucket(ctx context.Context, id *bucket.Identifier) (*bucket.Bucket, error) {
	handle := c.client.Bucket(id.Name)

	_, err := handle.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrBucketNotExist) {
			return nil, provider.NotFoundError
		}

		var ge *googleapi.Error
		if errors.As(err, &ge) {
			if ge.Code == http.StatusForbidden {
				return nil, provider.UnauthorizedError
			}
		}

		return nil, err
	}

	return &bucket.Bucket{
		Identifier: id,
		Config:     &bucket.Config{},
	}, nil
}

func (c *bucketClient) CreateBucket(ctx context.Context, id *bucket.Identifier, config *bucket.Config) error {
	return nil
}

func (c *bucketClient) GetBucketObject(ctx context.Context, id *bucketobject.Identifier) (*bucketobject.BucketObject, error) {
	// bucket, ok := id.Bucket.(*bucket.Identifier)
	// if !ok {
	// 	return nil, fmt.Errorf("incorrect type for bucket: %T", id.Bucket)
	// }
	//
	// handle := c.client.Bucket(bucket.Name)
	// objHandle := handle.Object(id.Name)
	// _, err := objHandle.Attrs(ctx)
	// if err != nil {
	// 	if errors.Is(err, storage.ErrObjectNotExist) {
	// 		return nil, provider.NotFoundError
	// 	}
	//
	// 	return nil, err
	// }

	return &bucketobject.BucketObject{}, nil
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

package aws

import (
	"fmt"

	"context"

	"github.com/alchematik/athanor/provider"

	"github.com/alchematik/athanor/gen/aws/v0.0.1/bucket"
)

type ClientRegistry struct {
	BucketClient bucket.Client
}

func (r *ClientRegistry) GetResource(ctx context.Context, identifier provider.Identifier) (*provider.Resource, error) {
	switch id := identifier.(type) {

	case *bucket.Identifier:
		return bucket.GetResource(ctx, r.BucketClient, id)

	default:
		return nil, fmt.Errorf("unrecognized identifier type: %T", identifier)
	}
}

func Schema() provider.Schema {
	return provider.Schema{
		Resources: map[string]provider.ResourceSchema{

			"bucket": bucket.ResourceSchema(),
		},
	}
}

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

func (r *ClientRegistry) GetResource(ctx context.Context, resourceType string, identifier []provider.FieldValue) (*provider.Resource, error) {
	switch resourceType {

	case "bucket":
		return bucket.GetResource(ctx, r.BucketClient, identifier)

	default:
		return nil, fmt.Errorf("unrecognized identifier type: %T", resourceType)
	}
}

func Schema() provider.Schema {
	return provider.Schema{
		Resources: map[string]provider.ResourceSchema{

			"bucket": bucket.ResourceSchema(),
		},
	}
}

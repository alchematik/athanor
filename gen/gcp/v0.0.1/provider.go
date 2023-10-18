package gcp

import (
	"fmt"

	"context"

	"github.com/alchematik/athanor/provider"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket_object"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/resource_policy"
)

type ClientRegistry struct {
	BucketClient bucket.Client

	BucketObjectClient bucket_object.Client

	ResourcePolicyClient resource_policy.Client
}

func (r *ClientRegistry) GetResource(ctx context.Context, identifier provider.Identifier) (*provider.Resource, error) {
	switch id := identifier.(type) {

	case *bucket.Identifier:
		return bucket.GetResource(ctx, r.BucketClient, id)

	case *bucket_object.Identifier:
		return bucket_object.GetResource(ctx, r.BucketObjectClient, id)

	case *resource_policy.Identifier:
		return resource_policy.GetResource(ctx, r.ResourcePolicyClient, id)

	default:
		return nil, fmt.Errorf("unrecognized identifier type: %T", identifier)
	}
}

func Schema() provider.Schema {
	return provider.Schema{
		Resources: map[string]provider.ResourceSchema{

			"bucket": bucket.ResourceSchema(),

			"bucket_object": bucket_object.ResourceSchema(),

			"resource_policy": resource_policy.ResourceSchema(),
		},
	}
}

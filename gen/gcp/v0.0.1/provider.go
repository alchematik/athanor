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

func (r *ClientRegistry) GetResource(ctx context.Context, resourceType string, identifier []provider.FieldValue) (*provider.Resource, error) {
	switch resourceType {

	case "bucket":
		return bucket.GetResource(ctx, r.BucketClient, identifier)

	case "bucket_object":
		return bucket_object.GetResource(ctx, r.BucketObjectClient, identifier)

	case "resource_policy":
		return resource_policy.GetResource(ctx, r.ResourcePolicyClient, identifier)

	default:
		return nil, fmt.Errorf("unrecognized identifier type: %T", resourceType)
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

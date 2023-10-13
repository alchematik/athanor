package gcp

import (
	"fmt"

	"context"

	"github.com/hashicorp/hcl/v2"

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

func (r *ClientRegistry) RegisterClient(client any) error {
	switch c := client.(type) {

	case bucket.Client:
		r.BucketClient = c

	case bucket_object.Client:
		r.BucketObjectClient = c

	case resource_policy.Client:
		r.ResourcePolicyClient = c

	default:
		return fmt.Errorf("unrecognized client type: %T", client)
	}

	return nil
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

func ParseIdentifierBlock(ctx *hcl.EvalContext, block *hcl.Block) (any, error) {
	switch block.Labels[1] {

	case "bucket":
		return bucket.ParseIdentifierBlock(ctx, block)

	case "bucket_object":
		return bucket_object.ParseIdentifierBlock(ctx, block)

	case "resource_policy":
		return resource_policy.ParseIdentifierBlock(ctx, block)

	default:
		return nil, fmt.Errorf("unknown block type: %s", block.Labels[1])
	}
}

func ParseOpBlock(ctx *hcl.EvalContext, block *hcl.Block) (provider.Operation, error) {
	switch block.Labels[1] {

	case "bucket":
		return bucket.ParseOpBlock(ctx, block)

	case "bucket_object":
		return bucket_object.ParseOpBlock(ctx, block)

	case "resource_policy":
		return resource_policy.ParseOpBlock(ctx, block)

	default:
		return nil, fmt.Errorf("unknown block type: %s", block.Labels[1])
	}
}

func ResourceNames() []string {
	return []string{

		"bucket",

		"bucket_object",

		"resource_policy",
	}
}

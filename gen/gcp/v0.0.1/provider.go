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

type Parser struct {
}

func (p *Parser) ParseIdentifierBlock(ctx *hcl.EvalContext, block *hcl.Block) (any, error) {
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

func (p *Parser) ParseOpBlock(ctx *hcl.EvalContext, block *hcl.Block) (provider.Operation, error) {
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

func (p *Parser) ResourceNames() []string {
	return []string{

		"bucket",

		"resource_policy",

		"bucket_object",
	}
}

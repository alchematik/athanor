package aws

import (
	"fmt"

	"context"

	"github.com/hashicorp/hcl/v2"

	"github.com/alchematik/athanor/provider"

	"github.com/alchematik/athanor/gen/aws/v0.0.1/bucket"
)

type ClientRegistry struct {
	BucketClient bucket.Client
}

func (r *ClientRegistry) RegisterClient(client any) error {
	switch c := client.(type) {

	case bucket.Client:
		r.BucketClient = c

	default:
		return fmt.Errorf("unrecognized client type: %T", client)
	}

	return nil
}

func (r *ClientRegistry) GetResource(ctx context.Context, identifier provider.Identifier) (*provider.Resource, error) {
	switch id := identifier.(type) {

	case *bucket.Identifier:
		return bucket.GetResource(ctx, r.BucketClient, id)

	default:
		return nil, fmt.Errorf("unrecognized identifier type: %T", identifier)
	}
}

func ParseIdentifierBlock(ctx *hcl.EvalContext, block *hcl.Block) (any, error) {
	switch block.Labels[1] {

	case "bucket":
		return bucket.ParseIdentifierBlock(ctx, block)

	default:
		return nil, fmt.Errorf("unknown block type: %s", block.Labels[1])
	}
}

func ParseOpBlock(ctx *hcl.EvalContext, block *hcl.Block) (provider.Operation, error) {
	switch block.Labels[1] {

	case "bucket":
		return bucket.ParseOpBlock(ctx, block)

	default:
		return nil, fmt.Errorf("unknown block type: %s", block.Labels[1])
	}
}

func ResourceNames() []string {
	return []string{

		"bucket",
	}
}

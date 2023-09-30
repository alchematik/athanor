package gcp

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/alchematik/athanor/testprovider/bucket"

	"github.com/alchematik/athanor/testprovider/bucket_object"

	"github.com/alchematik/athanor/testprovider/resource_policy"
)

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

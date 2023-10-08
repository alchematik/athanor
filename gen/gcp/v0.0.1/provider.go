package main

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/alchematik/athanor/operation"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket_object"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/resource_policy"
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

func ParseOpBlock(ctx *hcl.EvalContext, block *hcl.Block) (operation.Operation, error) {
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

package gcp

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"
	"github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket_object"
	"github.com/alchematik/athanor/gen/gcp/v0.0.1/resource_policy"
)

func ParseIdentifierBlock(evalCtx *hcl.EvalContext, block *hcl.Block) (any, error) {
	switch block.Labels[1] {
	case "bucket":
		return bucket.ParseIdentifierBlock(evalCtx, block)
	case "bucket_object":
		return bucket_object.ParseIdentifierBlock(evalCtx, block)
	case "resource_policy":
		return resource_policy.ParseIdentifierBlock(evalCtx, block)
	default:
		return nil, fmt.Errorf("unknown block type %v", block.Labels[1])
	}
}

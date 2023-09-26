package gcp

import (
	"fmt"

	"github.com/alchematik/athanor/testprovider/bucket"
	"github.com/alchematik/athanor/testprovider/bucket_object"
	"github.com/alchematik/athanor/testprovider/resource_policy"
	"github.com/hashicorp/hcl/v2"
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

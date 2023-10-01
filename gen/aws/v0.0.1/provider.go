package main

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/alchematik/athanor/gen/aws/v0.0.1/bucket"
)

func ParseIdentifierBlock(ctx *hcl.EvalContext, block *hcl.Block) (any, error) {
	switch block.Labels[1] {

	case "bucket":
		return bucket.ParseIdentifierBlock(ctx, block)

	default:
		return nil, fmt.Errorf("unknown block type: %s", block.Labels[1])
	}
}

func ResourceNames() []string {
	return []string{

		"bucket",
	}
}

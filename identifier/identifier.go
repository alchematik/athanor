package identifier

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type HCLIdentifier interface {
	CtyType() cty.Type
}

func AddIdentifierValueToEvalCtx(ctx *hcl.EvalContext, block *hcl.Block, value cty.Value) {
	blockType := block.Type
	provider := block.Labels[0]
	resource := block.Labels[1]
	name := block.Labels[2]

	m := ctx.Variables[blockType].AsValueMap()
	providerMap := m[provider].AsValueMap()
	resourceMap := providerMap[resource].AsValueMap()
	resourceMap[name] = value
	providerMap[resource] = cty.ObjectVal(resourceMap)
	m[provider] = cty.ObjectVal(providerMap)
	ctx.Variables[blockType] = cty.ObjectVal(m)
}

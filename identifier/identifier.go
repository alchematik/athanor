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

	typeMapVar, ok := ctx.Variables[blockType]
	if !ok {
		typeMapVar = cty.ObjectVal(map[string]cty.Value{
			"": cty.StringVal(""),
		})
	}
	typeMap := typeMapVar.AsValueMap()

	providerMapVar, ok := typeMap[provider]
	if !ok {
		providerMapVar = cty.ObjectVal(map[string]cty.Value{
			"": cty.StringVal(""),
		})
	}
	providerMap := providerMapVar.AsValueMap()

	resourceMapVar, ok := providerMap[resource]
	if !ok {
		resourceMapVar = cty.ObjectVal(map[string]cty.Value{
			"": cty.StringVal(""),
		})
	}
	resourceMap := resourceMapVar.AsValueMap()

	resourceMap[name] = value
	providerMap[resource] = cty.ObjectVal(resourceMap)
	typeMap[provider] = cty.ObjectVal(providerMap)
	ctx.Variables[blockType] = cty.ObjectVal(typeMap)
}

package parser

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type File struct {
	Path    string
	Content []byte
}

func AddIDToEvalCtx(ctx *hcl.EvalContext, provider, resource, name string, val cty.Value) {
	m := ctx.Variables["id"].AsValueMap()
	providerMap := m[provider].AsValueMap()
	resourceMap := providerMap[resource].AsValueMap()
	resourceMap[name] = val
	providerMap[resource] = cty.ObjectVal(resourceMap)
	m[provider] = cty.ObjectVal(providerMap)
}

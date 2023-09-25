package bucket

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

type HCLIdentifier struct {
	Account string `hcl:"account" cty:"account"`
	Region  string `hcl:"region" cty:"region"`
	Name    string `hcl:"name" cty:"name"`
}

type Identifier struct {
	Account string
	Region  string
	Name    string
}

func ParseIdentifierBlock(ctx *hcl.EvalContext, block *hcl.Block) (*Identifier, error) {
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "account"},
			{Name: "region"},
			{Name: "name"},
		},
	}
	content, diag := block.Body.Content(schema)
	if diag.HasErrors() {
		return nil, diag
	}

	var hclID HCLIdentifier
	if attr, ok := content.Attributes["name"]; ok {
		if diag := gohcl.DecodeExpression(attr.Expr, ctx, &hclID.Name); diag.HasErrors() {
			return nil, diag
		}
	}
	if attr, ok := content.Attributes["account"]; ok {
		if diag := gohcl.DecodeExpression(attr.Expr, ctx, &hclID.Account); diag.HasErrors() {
			return nil, diag
		}
	}

	if attr, ok := content.Attributes["region"]; ok {
		if diag := gohcl.DecodeExpression(attr.Expr, ctx, &hclID.Region); diag.HasErrors() {
			return nil, diag
		}
	}

	blockType := block.Type
	provider := block.Labels[0]
	resource := block.Labels[1]
	name := block.Labels[2]

	val, err := hclID.ToCtyValue()
	if err != nil {
		return nil, err
	}

	m := ctx.Variables[blockType].AsValueMap()
	providerMap := m[provider].AsValueMap()
	resourceMap := providerMap[resource].AsValueMap()
	resourceMap[name] = val
	providerMap[resource] = cty.ObjectVal(resourceMap)
	m[provider] = cty.ObjectVal(providerMap)
	ctx.Variables[blockType] = cty.ObjectVal(m)

	return hclID.ToIdentifier(), nil
}

func (id *HCLIdentifier) ToIdentifier() *Identifier {
	return &Identifier{
		Account: id.Account,
		Region:  id.Region,
		Name:    id.Name,
	}
}

func (id *HCLIdentifier) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{
		"account": cty.String,
		"name":    cty.String,
		"region":  cty.String,
	})
}

func (id *HCLIdentifier) ToCtyValue() (cty.Value, error) {
	return gocty.ToCtyValue(id, id.CtyType())
}

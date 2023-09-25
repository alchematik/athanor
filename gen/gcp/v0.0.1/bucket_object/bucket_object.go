package bucket_object

import (
	"github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

type HCLIdentifier struct {
	Bucket *bucket.HCLIdentifier `hcl:"bucket" cty:"bucket"`
	Name   string                `hcl:"name" cty:"name"`
}

type Identifier struct {
	Bucket *bucket.Identifier
	Name   string
}

func ParseIdentifierBlock(ctx *hcl.EvalContext, block *hcl.Block) (*Identifier, error) {
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "name"},
			{Name: "bucket"},
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
	if attr, ok := content.Attributes["bucket"]; ok {
		if diag := gohcl.DecodeExpression(attr.Expr, ctx, &hclID.Bucket); diag.HasErrors() {
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
		Name:   id.Name,
		Bucket: id.Bucket.ToIdentifier(),
	}
}

func (id *HCLIdentifier) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{
		"name":   cty.String,
		"bucket": id.Bucket.CtyType(),
	})
}

func (id *HCLIdentifier) ToCtyValue() (cty.Value, error) {
	return gocty.ToCtyValue(id, id.CtyType())
}

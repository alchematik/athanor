package resource_policy

import (
	"github.com/alchematik/athanor/identifier"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

type Identifier struct {
	Name     string
	Resource any
}

type HCLIdentifier struct {
	Name     string                   `hcl:"name" cty:"name"`
	Resource identifier.HCLIdentifier `hcl:"resource" cty:"resource"`
}

func ParseIdentifierBlock(ctx *hcl.EvalContext, block *hcl.Block) (*Identifier, error) {
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "name"},
			{Name: "resource"},
		},
	}
	content, diag := block.Body.Content(schema)
	if diag.HasErrors() {
		return nil, diag
	}

	var hclID HCLIdentifier
	if attr, ok := content.Attributes["name"]; ok {
		var name string
		if diag := gohcl.DecodeExpression(attr.Expr, ctx, &name); diag.HasErrors() {
			return nil, diag
		}

		hclID.Name = name
	}

	if attr, ok := content.Attributes["resource"]; ok {
		variable := attr.Expr.Variables()[0]
		part := variable.SimpleSplit().Rel[1].(hcl.TraverseAttr)
		resourceName := part.Name

		var hclResource identifier.HCLIdentifier
		switch resourceName {
		case "bucket":
			hclResource = &bucket.HCLIdentifier{}
		}

		if diag := gohcl.DecodeExpression(attr.Expr, ctx, hclResource); diag.HasErrors() {
			return nil, diag
		}

		hclID.Resource = hclResource
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
	var resource any
	switch t := id.Resource.(type) {
	case *bucket.HCLIdentifier:
		resource = t.ToIdentifier()
	}
	return &Identifier{
		Name:     id.Name,
		Resource: resource,
	}
}

func (id *HCLIdentifier) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{
		"name":     cty.String,
		"resource": id.Resource.CtyType(),
	})
}

func (id *HCLIdentifier) ToCtyValue() (cty.Value, error) {
	return gocty.ToCtyValue(id, id.CtyType())
}

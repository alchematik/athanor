package resource_policy

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/alchematik/athanor/identifier"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"
)

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
		hclID.Metadata.ResourceType = part.Name
		var val cty.Value
		if diag := gohcl.DecodeExpression(attr.Expr, ctx, &val); diag.HasErrors() {
			return nil, diag
		}
		hclID.Resource = val

	}

	val, err := hclID.ToCtyValue()
	if err != nil {
		return nil, err
	}

	identifier.AddIdentifierValueToEvalCtx(ctx, block, val)

	return hclID.ToIdentifier(), nil
}

// Identifier is the identifier for a resource_policy.
type Identifier struct {
	// Name is the name of the resource policy.
	Name string

	// Resource is the resource that the policy belongs to.
	Resource any
}

type HCLIdentifier struct {
	Metadata HCLIdentifierMetadata `cty:"metadata"`

	Name string `hcl:"name" cty:"name"`

	Resource cty.Value `hcl:"resource" cty:"resource"`
}

func (id *HCLIdentifier) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{

		"metadata": id.Metadata.CtyType(),

		"name":     cty.String,
		"resource": cty.DynamicPseudoType,
	})
}

func (id *HCLIdentifier) ToCtyValue() (cty.Value, error) {
	return gocty.ToCtyValue(id, id.CtyType())
}

func (id *HCLIdentifier) ToIdentifier() *Identifier {
	out := &Identifier{}

	out.Name = id.Name

	switch id.Metadata.ResourceType {
	case "bucket":
		var val bucket.HCLIdentifier
		if err := gocty.FromCtyValue(id.Resource, &val); err != nil {
			panic(err)
		}
		out.Resource = val.ToIdentifier()
	}

	return out
}

type HCLIdentifierMetadata struct {
	ResourceType string `cty:"resource_type"`
}

func (m HCLIdentifierMetadata) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{
		"resource_type": cty.String,
	})
}

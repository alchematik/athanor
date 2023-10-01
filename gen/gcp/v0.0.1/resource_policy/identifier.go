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
		var resource identifier.HCLIdentifier
		switch part.Name {

		case "bucket":
			resource = &bucket.HCLIdentifier{}

		}

		if diag := gohcl.DecodeExpression(attr.Expr, ctx, resource); diag.HasErrors() {
			return nil, diag
		}
		hclID.Resource = resource

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
	Name string `hcl:"name" cty:"name"`

	Resource identifier.HCLIdentifier `hcl:"resource" cty:"resource"`
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

func (id *HCLIdentifier) ToIdentifier() *Identifier {
	out := &Identifier{}

	out.Name = id.Name

	switch t := id.Resource.(type) {
	case *bucket.HCLIdentifier:
		out.Resource = t.ToIdentifier()
	}

	return out
}

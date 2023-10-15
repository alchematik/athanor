package resource_policy

import (
	"fmt"

	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/alchematik/athanor/provider"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"
	"github.com/alchematik/athanor/gen/gcp/v0.0.1/schema"
)

func ParseIdentifierBlock(ctx *hcl.EvalContext, block *hcl.Block) (*Identifier, error) {
	var hclAttrs []hcl.AttributeSchema
	for _, f := range schema.Schema["resource_policy"] {
		hclAttrs = append(hclAttrs, hcl.AttributeSchema{Name: f.Name})
	}
	content, diag := block.Body.Content(&hcl.BodySchema{Attributes: hclAttrs})
	if diag.HasErrors() {
		return nil, diag
	}

	var fvs []provider.FieldValue
	for _, f := range schema.Schema["resource_policy"] {
		if attr, ok := content.Attributes[f.Name]; ok {
			fv, err := provider.DecodeField(ctx, attr.Expr, f, schema.Schema)
			if err != nil {
				return nil, err
			}
			fvs = append(fvs, fv)
		}
	}

	fmt.Printf("resource_policy fvs: %+v\n", fvs)

	var hclID HCLIdentifier

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

	if attr, ok := content.Attributes["name"]; ok {

		var name string
		if diag := gohcl.DecodeExpression(attr.Expr, ctx, &name); diag.HasErrors() {
			return nil, diag
		}
		hclID.Name = name

	}

	val, err := hclID.ToCtyValue()
	if err != nil {
		return nil, err
	}

	provider.AddIdentifierValueToEvalCtx(ctx, block, val)

	return hclID.ToIdentifier(), nil
}

// Identifier is the identifier for a resource_policy.
type Identifier struct {
	// Resource is the resource that the policy belongs to.
	Resource any

	// Name is the name of the resource policy.
	Name string
}

func (id *Identifier) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("%s", id.Resource))

	parts = append(parts, "resource_policy", fmt.Sprintf("%s", id.Name))

	return strings.Join(parts, "/")
}

type HCLIdentifier struct {
	Metadata HCLIdentifierMetadata `cty:"metadata"`

	Resource cty.Value `hcl:"resource" cty:"resource"`

	Name string `hcl:"name" cty:"name"`
}

func (id *HCLIdentifier) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{

		"metadata": id.Metadata.CtyType(),

		"resource": cty.DynamicPseudoType,
		"name":     cty.String,
	})
}

func (id *HCLIdentifier) ToCtyValue() (cty.Value, error) {
	return gocty.ToCtyValue(id, id.CtyType())
}

func (id *HCLIdentifier) ToIdentifier() *Identifier {
	out := &Identifier{}

	switch id.Metadata.ResourceType {
	case "bucket":
		var val bucket.HCLIdentifier
		if err := gocty.FromCtyValue(id.Resource, &val); err != nil {
			panic(err)
		}
		out.Resource = val.ToIdentifier()
	}

	out.Name = id.Name

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

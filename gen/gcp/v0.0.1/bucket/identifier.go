package bucket

import (
	"fmt"

	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/schema"
	"github.com/alchematik/athanor/provider"
)

func ParseIdentifierBlock(ctx *hcl.EvalContext, block *hcl.Block) (*Identifier, error) {
	var hclAttrs []hcl.AttributeSchema
	for _, f := range schema.Schema["bucket"] {
		hclAttrs = append(hclAttrs, hcl.AttributeSchema{Name: f.Name})
	}
	content, diag := block.Body.Content(&hcl.BodySchema{Attributes: hclAttrs})
	if diag.HasErrors() {
		return nil, diag
	}

	var fvs []provider.FieldValue
	for _, f := range schema.Schema["bucket"] {
		if attr, ok := content.Attributes[f.Name]; ok {
			fv, err := provider.DecodeField(ctx, attr.Expr, f, schema.Schema)
			if err != nil {
				return nil, err
			}
			fvs = append(fvs, fv)
		}
	}

	fmt.Printf("bucket fvs: %+v\n", fvs)

	var hclID HCLIdentifier
	if attr, ok := content.Attributes["project"]; ok {

		var project string
		if diag := gohcl.DecodeExpression(attr.Expr, ctx, &project); diag.HasErrors() {
			return nil, diag
		}
		hclID.Project = project

	}

	if attr, ok := content.Attributes["region"]; ok {

		var region string
		if diag := gohcl.DecodeExpression(attr.Expr, ctx, &region); diag.HasErrors() {
			return nil, diag
		}
		hclID.Region = region

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

// Identifier is the identifier for a bucket.
type Identifier struct {
	// Project is the project that the bucket belongs to.
	Project string

	// Region is the region that the bucket belongs in.
	Region string

	// Name is the name of the bucket.
	Name string
}

func (id *Identifier) String() string {
	var parts []string

	parts = append(parts, "gcp", "v0.0.1")

	parts = append(parts, fmt.Sprintf("%s", id.Project))

	parts = append(parts, fmt.Sprintf("%s", id.Region))

	parts = append(parts, "bucket", fmt.Sprintf("%s", id.Name))

	return strings.Join(parts, "/")
}

type HCLIdentifier struct {
	Project string `hcl:"project" cty:"project"`

	Region string `hcl:"region" cty:"region"`

	Name string `hcl:"name" cty:"name"`
}

func (id *HCLIdentifier) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{

		"project": cty.String,
		"region":  cty.String,
		"name":    cty.String,
	})
}

func (id *HCLIdentifier) ToCtyValue() (cty.Value, error) {
	return gocty.ToCtyValue(id, id.CtyType())
}

func (id *HCLIdentifier) ToIdentifier() *Identifier {
	out := &Identifier{}

	out.Project = id.Project

	out.Region = id.Region

	out.Name = id.Name

	return out
}

type HCLIdentifierMetadata struct {
}

func (m HCLIdentifierMetadata) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{})
}

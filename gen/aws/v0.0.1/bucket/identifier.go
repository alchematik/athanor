package bucket

import (
	"fmt"

	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/alchematik/athanor/identifier"
)

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

	if attr, ok := content.Attributes["account"]; ok {

		var account string
		if diag := gohcl.DecodeExpression(attr.Expr, ctx, &account); diag.HasErrors() {
			return nil, diag
		}
		hclID.Account = account

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

	identifier.AddIdentifierValueToEvalCtx(ctx, block, val)

	return hclID.ToIdentifier(), nil
}

// Identifier is the identifier for a bucket.
type Identifier struct {
	// Account is the account that the bucket belongs to.
	Account string

	// Region is the region that the bucket belongs in.
	Region string

	// Name is the name of the bucket.
	Name string
}

func (id *Identifier) String() string {
	var parts []string

	parts = append(parts, "aws", "v0.0.1")

	parts = append(parts, fmt.Sprintf("%s", id.Account))

	parts = append(parts, fmt.Sprintf("%s", id.Region))

	parts = append(parts, "bucket", fmt.Sprintf("%s", id.Name))

	return strings.Join(parts, "/")
}

type HCLIdentifier struct {
	Account string `hcl:"account" cty:"account"`

	Region string `hcl:"region" cty:"region"`

	Name string `hcl:"name" cty:"name"`
}

func (id *HCLIdentifier) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{

		"account": cty.String,
		"region":  cty.String,
		"name":    cty.String,
	})
}

func (id *HCLIdentifier) ToCtyValue() (cty.Value, error) {
	return gocty.ToCtyValue(id, id.CtyType())
}

func (id *HCLIdentifier) ToIdentifier() *Identifier {
	out := &Identifier{}

	out.Account = id.Account

	out.Region = id.Region

	out.Name = id.Name

	return out
}

type HCLIdentifierMetadata struct {
}

func (m HCLIdentifierMetadata) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{})
}

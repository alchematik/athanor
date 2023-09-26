package bucket_object

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/alchematik/athanor/identifier"

	"github.com/alchematik/athanor/testprovider/bucket"
)

func ParseIdentifierBlock(ctx *hcl.EvalContext, block *hcl.Block) (*Identifier, error) {
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "bucket"},
			{Name: "name"},
		},
	}
	content, diag := block.Body.Content(schema)
	if diag.HasErrors() {
		return nil, diag
	}

	var hclID HCLIdentifier

	if attr, ok := content.Attributes["bucket"]; ok {

		var bucket *bucket.HCLIdentifier
		if diag := gohcl.DecodeExpression(attr.Expr, ctx, &bucket); diag.HasErrors() {
			return nil, diag
		}
		hclID.Bucket = bucket

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

// Identifier is the identifier for a bucket_object.
type Identifier struct {
	// Bucket is the bucket that the object belongs to.
	Bucket *bucket.Identifier

	// Name is the name of the bucket_object.
	Name string
}

type HCLIdentifier struct {
	Bucket *bucket.HCLIdentifier `hcl:"bucket" cty:"bucket"`

	Name string `hcl:"name" cty:"name"`
}

func (id *HCLIdentifier) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{
		"bucket": id.Bucket.CtyType(),
		"name":   cty.String,
	})
}

func (id *HCLIdentifier) ToCtyValue() (cty.Value, error) {
	return gocty.ToCtyValue(id, id.CtyType())
}

func (id *HCLIdentifier) ToIdentifier() *Identifier {
	out := &Identifier{}

	out.Bucket = id.Bucket.ToIdentifier()

	out.Name = id.Name

	return out
}

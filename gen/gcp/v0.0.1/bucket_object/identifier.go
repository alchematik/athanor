package bucket_object

import (
	"fmt"

	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/hcl/v2"

	"github.com/alchematik/athanor/provider"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"
	"github.com/alchematik/athanor/gen/gcp/v0.0.1/schema"
)

func ParseIdentifierBlock(ctx *hcl.EvalContext, block *hcl.Block) (*Identifier, error) {
	var hclAttrs []hcl.AttributeSchema
	for _, f := range schema.Schema["bucket_object"] {
		hclAttrs = append(hclAttrs, hcl.AttributeSchema{Name: f.Name})
	}
	content, diag := block.Body.Content(&hcl.BodySchema{Attributes: hclAttrs})
	if diag.HasErrors() {
		return nil, diag
	}

	var fvs []provider.FieldValue
	for _, f := range schema.Schema["bucket_object"] {
		if attr, ok := content.Attributes[f.Name]; ok {
			fv, err := provider.DecodeField(ctx, attr.Expr, f, schema.Schema)
			if err != nil {
				return nil, err
			}
			fvs = append(fvs, fv)
		}
	}

	fmt.Printf("bucket_object fvs: %+v\n", fvs)

	// val, err := hclID.ToCtyValue()
	// if err != nil {
	// 	return nil, err
	// }
	val, err := provider.FieldValuesToCtyValue(fvs)
	if err != nil {
		return nil, err
	}

	provider.AddIdentifierValueToEvalCtx(ctx, block, val)

	return nil, nil
}

// Identifier is the identifier for a bucket_object.
type Identifier struct {
	// Bucket is the bucket that the object belongs to.
	Bucket *bucket.Identifier

	// Name is the name of the bucket_object.
	Name string
}

func (id *Identifier) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("%s", id.Bucket))

	parts = append(parts, "bucket_object", fmt.Sprintf("%s", id.Name))

	return strings.Join(parts, "/")
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

type HCLIdentifierMetadata struct {
}

func (m HCLIdentifierMetadata) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{})
}

package bucket_object

import (
	"fmt"
	"github.com/alchematik/athanor/provider"

	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/hashicorp/hcl/v2"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/schema"
)

func ParseOpBlock(ctx *hcl.EvalContext, block *hcl.Block) (provider.Operation, error) {
	s := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "id"},
			{Name: "version"},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type: "config",
			},
		},
	}
	content, diag := block.Body.Content(s)
	if diag.HasErrors() {
		return nil, diag
	}

	var op HCLOp
	idAttr := content.Attributes["id"]

	ivf, err := provider.DecodeField(ctx, idAttr.Expr, provider.Field{Name: "id", Type: "bucket_object"}, schema.Schema)
	if err != nil {
		return nil, err
	}

	fmt.Printf("bucket_object op id: %+v\n", ivf)

	versionAttr := content.Attributes["version"]
	var version string
	if diag := gohcl.DecodeExpression(versionAttr.Expr, ctx, &version); diag.HasErrors() {
		return nil, diag
	}

	op.Version = version

	op.Type = block.Type

	for _, b := range content.Blocks {
		if b.Type == "config" {
			var attrs []hcl.AttributeSchema
			for _, f := range schema.ConfigSchema["bucket_object"] {
				attrs = append(attrs, hcl.AttributeSchema{Name: f.Name})
			}

			configContent, diag := b.Body.Content(&hcl.BodySchema{Attributes: attrs})
			if diag.HasErrors() {
				return nil, diag
			}

			var fvs []provider.FieldValue
			for _, f := range schema.ConfigSchema["bucket_object"] {
				if attr, ok := configContent.Attributes[f.Name]; ok {
					fv, err := provider.DecodeField(ctx, attr.Expr, f, schema.Schema)
					if err != nil {
						return nil, err
					}
					fvs = append(fvs, fv)
				}
			}

			fmt.Printf("bucket_object config: %+v\n", fvs)
		}
	}

	return nil, nil
}

type Op struct {
	Type       string
	Identifier *Identifier
	Version    string
	Config     Config
}

func (o *Op) ForIdentifier() provider.Identifier {
	return o.Identifier
}

func (o *Op) ForVersion() string {
	return o.Version
}

func (o *Op) Apply(r *provider.Resource) {
	r.State = provider.ResourceStateExists
	r.Identifier = o.Identifier
	r.Config = o.Config
}

type Config struct {
	Contents string
}

type HCLOp struct {
	Type          string         `hcl:"type" cty:"type"`
	HCLIdentifier *HCLIdentifier `hcl:"id" cty:"id"`
	Version       string         `hcl:"version" cty:"version"`
	HCLConfig     HCLConfig      `hcl:"config" cty:"config"`
}

func (op *HCLOp) ToOp() provider.Operation {
	return &Op{
		Type:       op.Type,
		Identifier: op.HCLIdentifier.ToIdentifier(),
		Version:    op.Version,
		Config:     op.HCLConfig.ToConfig(),
	}
}

type HCLConfig struct {
	Contents string `hcl:"contents" cty:"contents"`
}

func (c HCLConfig) ToConfig() Config {
	out := Config{}

	out.Contents = c.Contents

	return out
}

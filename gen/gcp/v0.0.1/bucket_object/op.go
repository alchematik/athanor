package bucket_object

import (
	"github.com/alchematik/athanor/provider"

	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/hashicorp/hcl/v2"
)

func ParseOpBlock(ctx *hcl.EvalContext, block *hcl.Block) (provider.Operation, error) {
	schema := &hcl.BodySchema{
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
	content, diag := block.Body.Content(schema)
	if diag.HasErrors() {
		return nil, diag
	}

	var op HCLOp
	idAttr := content.Attributes["id"]
	var id HCLIdentifier
	if diag := gohcl.DecodeExpression(idAttr.Expr, ctx, &id); diag.HasErrors() {
		return nil, diag
	}

	op.HCLIdentifier = &id

	versionAttr := content.Attributes["version"]
	var version string
	if diag := gohcl.DecodeExpression(versionAttr.Expr, ctx, &version); diag.HasErrors() {
		return nil, diag
	}

	op.Version = version

	op.Type = block.Type

	for _, b := range content.Blocks {
		if b.Type == "config" {
			var hclConfig HCLConfig
			configSchema := &hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{

					{Name: "contents"},
				},
			}
			configContent, diag := b.Body.Content(configSchema)
			if diag.HasErrors() {
				return nil, diag
			}

			if attr, ok := configContent.Attributes["contents"]; ok {
				var contents string
				if diag := gohcl.DecodeExpression(attr.Expr, ctx, &contents); diag.HasErrors() {
					return nil, diag
				}

				hclConfig.Contents = contents
			}

			op.HCLConfig = hclConfig
		}
	}

	return op.ToOp(), nil
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

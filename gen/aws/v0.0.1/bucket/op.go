package bucket

import (
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/hashicorp/hcl/v2"
)

func ParseOpBlock(ctx *hcl.EvalContext, block *hcl.Block) (*Op, error) {
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "id"},
			{Name: "config"},
			{Name: "version"},
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

	if configAttr, ok := content.Attributes["config"]; ok {
		var hclConfig HCLConfig
		if diag := gohcl.DecodeExpression(configAttr.Expr, ctx, &hclConfig); diag.HasErrors() {
			return nil, diag
		}

		op.HCLConfig = hclConfig
	}

	return op.ToOp(), nil
}

type Op struct {
	Type       string
	Identifier *Identifier
	Version    string
	Config     Config
}

type Config struct {
}

type HCLOp struct {
	Type          string         `hcl:"type" cty:"type"`
	HCLIdentifier *HCLIdentifier `hcl:"id" cty:"id"`
	Version       string         `hcl:"version" cty:"version"`
	HCLConfig     HCLConfig      `hcl:"config" cty:"config"`
}

func (op *HCLOp) ToOp() *Op {
	return &Op{
		Type:       op.Type,
		Identifier: op.HCLIdentifier.ToIdentifier(),
		Version:    op.Version,
		Config:     op.HCLConfig.ToConfig(),
	}
}

type HCLConfig struct {
}

func (c HCLConfig) ToConfig() Config {
	out := Config{}

	return out
}

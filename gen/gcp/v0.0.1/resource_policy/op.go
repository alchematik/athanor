package resource_policy

import (
	"fmt"
	"github.com/alchematik/athanor/provider"

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
	ivf, err := provider.DecodeField(ctx, idAttr.Expr, provider.Field{Name: "id", Type: "resource_policy"}, schema.Schema)
	if err != nil {
		return nil, err
	}

	fmt.Printf("resource_policy op id: %+v\n", ivf)

	// op.HCLIdentifier = &id

	versionAttr := content.Attributes["version"]
	vfv, err := provider.DecodeField(ctx, versionAttr.Expr, provider.Field{Name: "version", Type: "string"}, schema.Schema)
	if err != nil {
		return nil, err
	}

	op.Version = vfv.Value.(string)

	fmt.Printf("resource_policy op version: %+v\n", vfv)

	op.Type = block.Type

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
}

func (c HCLConfig) ToConfig() Config {
	out := Config{}

	return out
}

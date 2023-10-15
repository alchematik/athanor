package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

var (
	NotFoundError = errors.New("not found")
)

type Provider struct {
	clients ClientRegistry
	parser  Parser
}

type ClientRegistry interface {
	GetResource(context.Context, Identifier) (*Resource, error)
}

type Parser interface {
	ParseIdentifierBlock(ctx *hcl.EvalContext, block *hcl.Block) (any, error)
	ParseOpBlock(ctx *hcl.EvalContext, block *hcl.Block) (Operation, error)
	ResourceNames() []string
}

func New(r ClientRegistry, p Parser) *Provider {
	return &Provider{
		clients: r,
		parser:  p,
	}
}

func (p *Provider) ParseIdentifierBlock(ctx *hcl.EvalContext, block *hcl.Block) (any, error) {
	return p.parser.ParseIdentifierBlock(ctx, block)
}

func (p *Provider) ParseOpBlock(ctx *hcl.EvalContext, block *hcl.Block) (Operation, error) {
	return p.parser.ParseOpBlock(ctx, block)
}

func (p *Provider) ResourceNames() []string {
	return p.parser.ResourceNames()
}

func (p *Provider) GetResource(ctx context.Context, id Identifier) (*Resource, error) {
	return p.clients.GetResource(ctx, id)
}

/*

- expect NewProvider() (ClientRegistry, error) function to be defined.

*/

type ResourceState string

const (
	ResourceStateExists    = "exists"
	ResourceStateNotExists = "not_exists"
)

type Operation interface {
	ForIdentifier() Identifier
	ForVersion() string
	Apply(*Resource)
}

type Identifier interface {
	String() string
}

type Resource struct {
	State      ResourceState
	Identifier Identifier
	Config     any
}

func AddIdentifierValueToEvalCtx(ctx *hcl.EvalContext, block *hcl.Block, value cty.Value) {
	blockType := block.Type
	provider := block.Labels[0]
	resource := block.Labels[1]
	name := block.Labels[2]

	typeMapVar, ok := ctx.Variables[blockType]
	if !ok {
		typeMapVar = cty.ObjectVal(map[string]cty.Value{
			"": cty.StringVal(""),
		})
	}
	typeMap := typeMapVar.AsValueMap()

	providerMapVar, ok := typeMap[provider]
	if !ok {
		providerMapVar = cty.ObjectVal(map[string]cty.Value{
			"": cty.StringVal(""),
		})
	}
	providerMap := providerMapVar.AsValueMap()

	resourceMapVar, ok := providerMap[resource]
	if !ok {
		resourceMapVar = cty.ObjectVal(map[string]cty.Value{
			"": cty.StringVal(""),
		})
	}
	resourceMap := resourceMapVar.AsValueMap()

	resourceMap[name] = value
	providerMap[resource] = cty.ObjectVal(resourceMap)
	typeMap[provider] = cty.ObjectVal(providerMap)
	ctx.Variables[blockType] = cty.ObjectVal(typeMap)
}

type Field struct {
	Name  string
	Type  string
	Oneof []string
}

type FieldValue struct {
	Name  string
	Type  string
	Value any
}

type Schema map[string][]Field

func DecodeField(ctx *hcl.EvalContext, expr hcl.Expression, f Field, s Schema) (FieldValue, error) {
	fv := FieldValue{
		Name: f.Name,
		Type: f.Type,
	}
	switch f.Type {
	case "string":
		var val string
		if diag := gohcl.DecodeExpression(expr, ctx, &val); diag.HasErrors() {
			return FieldValue{}, diag
		}
		fv.Value = val
	case "oneof":
		variable := expr.Variables()[0]
		subtype := variable.SimpleSplit().Rel[1].(hcl.TraverseAttr).Name
		sub := s[subtype]
		var match bool
		for _, o := range f.Oneof {
			if subtype == o {
				match = true
			}
		}
		if !match {
			return FieldValue{}, fmt.Errorf("not oneof type: %v", subtype)
		}

		val, diag := expr.Value(ctx)
		if diag.HasErrors() {
			return FieldValue{}, diag
		}

		m := val.AsValueMap()
		var vals []FieldValue
		for _, f := range sub {
			var v any
			switch f.Type {
			case "string":
				v = m[f.Name].AsString()
			}

			fv := FieldValue{
				Type:  f.Type,
				Name:  f.Name,
				Value: v,
			}
			vals = append(vals, fv)
		}

		fv.Value = vals

	default:
		// TODO: Will this work with nested maps?
		val, diag := expr.Value(ctx)
		if diag.HasErrors() {
			return FieldValue{}, diag
		}

		m := val.AsValueMap()
		var vals []FieldValue
		sub := s[f.Type]
		for _, f := range sub {
			var v any
			switch f.Type {
			case "string":
				v = m[f.Name].AsString()
			}

			fv := FieldValue{
				Type:  f.Type,
				Name:  f.Name,
				Value: v,
			}
			vals = append(vals, fv)
		}

		fv.Value = vals
	}

	return fv, nil
}

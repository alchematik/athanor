package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/hcl/v2"
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

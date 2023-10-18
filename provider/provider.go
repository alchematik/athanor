package provider

import (
	"context"
	"errors"
	"fmt"
	"net/rpc"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

var (
	NotFoundError = errors.New("not found")
)

type ClientRegistry interface {
	GetResource(context.Context, Identifier) (*Resource, error)
}

type SchemaProvider interface {
	ResourceNames() []string
	IdentifierSchema() map[string][]Field
	ConfigSchema() map[string][]Field
}

type ProviderRPCClient struct {
	client *rpc.Client
}

func (p *ProviderRPCClient) ResourceNames() ([]string, error) {
	var res []string
	err := p.client.Call("Plugin.ResourceNames", new(interface{}), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (p *ProviderRPCClient) IdentifierSchema() (map[string][]Field, error) {
	var res map[string][]Field
	if err := p.client.Call("Plugin.IdentifierSchema", new(interface{}), &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (p *ProviderRPCClient) ConfigSchema() (map[string][]Field, error) {
	var res map[string][]Field
	if err := p.client.Call("Plugin.ConfigSchema", new(interface{}), &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (p *ProviderRPCClient) GetResource(id Identifier) (*Resource, error) {
	var res Resource
	if err := p.client.Call("Plugin.GetResource", id, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

type ProviderRPCServer struct {
	ClientRegistry ClientRegistry
	SchemaProvider SchemaProvider
}

func (s *ProviderRPCServer) ResourceNames(args any, resp *[]string) error {
	*resp = s.SchemaProvider.ResourceNames()
	return nil
}

func (s *ProviderRPCServer) IdentifierSchema(args any, resp *map[string][]Field) error {
	*resp = s.SchemaProvider.IdentifierSchema()
	return nil
}

func (s *ProviderRPCServer) ConfigSchema(args any, resp *map[string][]Field) error {
	*resp = s.SchemaProvider.ConfigSchema()
	return nil
}

func (s *ProviderRPCServer) GetResource(id Identifier, resp *Resource) error {
	// TODO: pass in context?
	r, err := s.ClientRegistry.GetResource(context.Background(), id)
	if err != nil {
		return err
	}

	resp = r
	return nil
}

type ProviderPlugin struct {
	ClientRegistry ClientRegistry
	SchemaProvider SchemaProvider
}

func (p *ProviderPlugin) Server(*plugin.MuxBroker) (any, error) {
	return &ProviderRPCServer{
		ClientRegistry: p.ClientRegistry,
		SchemaProvider: p.SchemaProvider,
	}, nil
}

func (p *ProviderPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (any, error) {
	return &ProviderRPCClient{client: c}, nil
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
	Name     string
	Type     string
	Value    any
	Metadata map[string]string
}

type Schema map[string][]Field

func DecodeField(ctx *hcl.EvalContext, expr hcl.Expression, f Field, s Schema) (FieldValue, error) {
	fv := FieldValue{
		Name:     f.Name,
		Type:     f.Type,
		Metadata: map[string]string{},
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
		fv.Metadata["oneof_type"] = subtype
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
		vals := []FieldValue{
			// {
			// 	Name:  fmt.Sprintf("%s_metadata", f.Name),
			// 	Type:  "string",
			// 	Value: subtype,
			// },
		}
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
		vals, err := parseValues(m, s[f.Type], s)
		if err != nil {
			return FieldValue{}, err
		}

		fv.Value = vals
	}

	return fv, nil
}

func FieldValuesToCtyType(fields []FieldValue) cty.Type {
	m := map[string]cty.Type{}
	for _, f := range fields {
		m[fmt.Sprintf("%s_metadata", f.Name)] = cty.Map(cty.String)

		switch f.Type {
		case "string":
			m[f.Name] = cty.String
		default:
			vals, ok := f.Value.([]FieldValue)
			if !ok {
				panic(fmt.Sprintf("expected list of field values, got %T", f.Value))
			}
			t := FieldValuesToCtyType(vals)
			m[f.Name] = t
		}
	}

	return cty.Object(m)
}

func FieldValuesToCtyValue(fields []FieldValue) (cty.Value, error) {
	t := FieldValuesToCtyType(fields)
	m := fieldValuesToMap(fields)
	return gocty.ToCtyValue(m, t)
}

func fieldValuesToMap(fields []FieldValue) map[string]any {
	m := map[string]any{}
	for _, f := range fields {
		metadata := map[string]string{}
		for k, v := range f.Metadata {
			metadata[k] = v
		}
		m[fmt.Sprintf("%s_metadata", f.Name)] = metadata
		switch f.Type {
		case "string":
			m[f.Name] = f.Value
		default:
			vals, ok := f.Value.([]FieldValue)
			if !ok {
				panic(fmt.Sprintf("expected list of field values, got %T", f.Value))
			}

			m[f.Name] = fieldValuesToMap(vals)
		}
	}

	return m
}

func parseValues(m map[string]cty.Value, fields []Field, schema map[string][]Field) ([]FieldValue, error) {
	var vals []FieldValue
	for _, f := range fields {
		switch f.Type {
		case "string":
			vals = append(vals, FieldValue{
				Type:  f.Type,
				Name:  f.Name,
				Value: m[f.Name].AsString(),
			})
		case "oneof":
			metadataValue := m[fmt.Sprintf("%s_metadata", f.Name)].AsValueMap()
			metadata := map[string]string{}
			for k, v := range metadataValue {
				metadata[k] = v.AsString()
			}
			oneofType := metadata["oneof_type"]
			subVals, err := parseValues(m[f.Name].AsValueMap(), schema[oneofType], schema)
			if err != nil {
				return nil, err
			}
			vals = append(vals, FieldValue{
				Type:     f.Type,
				Name:     f.Name,
				Value:    subVals,
				Metadata: metadata,
			})
		default:
			subvals, err := parseValues(m[f.Name].AsValueMap(), schema[f.Name], schema)
			if err != nil {
				return nil, err
			}
			vals = append(vals, FieldValue{
				Type:  f.Type,
				Name:  f.Name,
				Value: subvals,
			})
		}
	}

	return vals, nil
}

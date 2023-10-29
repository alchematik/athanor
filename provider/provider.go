package provider

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net/rpc"
	"strings"

	// "github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

var (
	NotFoundError     = errors.New("not found")
	UnauthorizedError = errors.New("unauthorized")
)

func init() {
	gob.Register([]FieldValue{})
}

type ClientRegistry interface {
	GetResource(context.Context, string, []FieldValue) (*Resource, error)
}

type ProviderRPCClient struct {
	client *rpc.Client
}

func (p *ProviderRPCClient) Schema() (Schema, error) {
	var res Schema
	err := p.client.Call("Plugin.Schema", new(interface{}), &res)
	if err != nil {
		return Schema{}, err
	}

	return res, nil
}

type GetResurceInput struct {
	IdentifierFields []FieldValue
	ResourceType     string
}

func (p *ProviderRPCClient) GetResource(input GetResurceInput) (*Resource, error) {
	var res Resource
	if err := p.client.Call("Plugin.GetResource", input, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

type ProviderRPCServer struct {
	ClientRegistry ClientRegistry
	ProviderSchema Schema
}

func (s *ProviderRPCServer) Schema(args any, resp *Schema) error {
	*resp = s.ProviderSchema
	return nil
}

func (s *ProviderRPCServer) GetResource(input GetResurceInput, resp *Resource) error {
	// TODO: pass in context?
	r, err := s.ClientRegistry.GetResource(context.Background(), input.ResourceType, input.IdentifierFields)
	if err != nil {
		return err
	}

	*resp = *r
	return nil
}

type ProviderPlugin struct {
	ClientRegistry ClientRegistry
	Schema         Schema
}

func (p *ProviderPlugin) Server(*plugin.MuxBroker) (any, error) {
	return &ProviderRPCServer{
		ClientRegistry: p.ClientRegistry,
		ProviderSchema: p.Schema,
	}, nil
}

func (p *ProviderPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (any, error) {
	return &ProviderRPCClient{client: c}, nil
}

type ResourceState string

const (
	ResourceStateExists    = "exists"
	ResourceStateNotExists = "not_exists"
)

type Identifier interface {
	String() string
}

type Resource struct {
	State            ResourceState
	Provider         string
	Type             string
	IdentifierFields []FieldValue
	ConfigFields     []FieldValue
	DependsOn        []Resource
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
	Name string
	Type string
}

type FieldValue struct {
	Name     string
	Type     string
	Value    any
	Metadata map[string]string
}

type Schema struct {
	Resources map[string]ResourceSchema
}

type ResourceSchema struct {
	IdentifierFields []Field
	ConfigFields     []Field
	DependsOn        []string
}

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
	case "identifier":
		variable := expr.Variables()[0]
		subtype := variable.SimpleSplit().Rel[1].(hcl.TraverseAttr).Name
		sub := s.Resources[subtype].IdentifierFields
		fv.Metadata["identifier_type"] = subtype
		val, diag := expr.Value(ctx)
		if diag.HasErrors() {
			return FieldValue{}, diag
		}

		m := val.AsValueMap()
		vals, err := parseValues(m, sub, s)
		if err != nil {
			return FieldValue{}, err
		}
		fv.Value = vals

	default:
		return FieldValue{}, fmt.Errorf("unsupported type: %q", f.Type)
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
		case "identifier":
			vals, ok := f.Value.([]FieldValue)
			if !ok {
				panic(fmt.Sprintf("expected list of field values, got %T", f.Value))
			}
			t := FieldValuesToCtyType(vals)
			m[f.Name] = t
		default:
			// TODO: Return error.
			panic(fmt.Sprintf("unexpected type %q", f.Type))
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
		case "identifier":
			vals, ok := f.Value.([]FieldValue)
			if !ok {
				panic(fmt.Sprintf("expected list of field values, got %T", f.Value))
			}

			m[f.Name] = fieldValuesToMap(vals)
		default:
			panic(fmt.Sprintf("unexpected type: %q", f.Type))
		}
	}

	return m
}

func parseValues(m map[string]cty.Value, fields []Field, schema Schema) ([]FieldValue, error) {
	var vals []FieldValue
	for _, f := range fields {
		switch f.Type {
		case "string":
			vals = append(vals, FieldValue{
				Type:  f.Type,
				Name:  f.Name,
				Value: m[f.Name].AsString(),
			})
		case "identifier":
			metadataValue := m[fmt.Sprintf("%s_metadata", f.Name)].AsValueMap()
			metadata := map[string]string{}
			for k, v := range metadataValue {
				metadata[k] = v.AsString()
			}
			oneofType := metadata["identifier_type"]
			subVals, err := parseValues(m[f.Name].AsValueMap(), schema.Resources[oneofType].IdentifierFields, schema)
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
			return nil, fmt.Errorf("unexpected type: %q", f.Type)
		}
	}

	return vals, nil
}

type Operation struct {
	ResourceType     string
	Provider         string
	IdentifierFields []FieldValue
	ConfigFields     []FieldValue
	Action           string
	Version          string
	DependsOn        [][]FieldValue
}

type State struct {
	Resources map[string]Resource
}

func (s State) Apply(op Operation) error {
	var parts []string
	for _, f := range op.IdentifierFields {
		parts = append(parts, fieldValueToString(f))
	}
	id := fmt.Sprintf("%s://%s", op.ResourceType, strings.Join(parts, "/"))
	switch op.Action {
	case "create":
		r, ok := s.Resources[id]
		if ok {
			if r.State != ResourceStateNotExists {
				return fmt.Errorf("resource already exists")
			}
		}

		s.Resources[id] = Resource{
			Provider:         op.Provider,
			Type:             op.ResourceType,
			IdentifierFields: op.IdentifierFields,
			ConfigFields:     op.ConfigFields,
			State:            ResourceStateExists,
		}

		return nil
	default:
		return fmt.Errorf("unsupported action: %q", op.Action)
	}
}

func fieldValueToString(fv FieldValue) string {
	switch fv.Type {
	case "string":
		return strings.Join([]string{fv.Name, fv.Value.(string)}, "/")
	case "identifier":
		var parts []string
		for _, v := range fv.Value.([]FieldValue) {
			parts = append(parts, fieldValueToString(v))
		}
		return strings.Join(parts, "/")
	default:
		panic(fmt.Sprintf("unexpected type: %q", fv.Type))
	}
}

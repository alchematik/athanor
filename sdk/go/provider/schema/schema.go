package schema

import (
	providerpb "github.com/alchematik/athanor/internal/gen/go/proto/provider/v1"
)

const (
	FieldTypeEmpty  FieldType = ""
	FieldTypeString FieldType = "string"
	FieldTypeStruct FieldType = "struct"
)

type Schema struct {
	Name      string
	Version   string
	Resources []ResourceSchema
}

func (s Schema) ToProto() *providerpb.Schema {
	resources := make([]*providerpb.ResourceSchema, len(s.Resources))
	for i, r := range s.Resources {
		resources[i] = r.ToProto()
	}

	return &providerpb.Schema{
		Name:      s.Name,
		Version:   s.Version,
		Resources: resources,
	}
}

type ResourceSchema struct {
	Type       string
	Identifier FieldSchema
	Config     FieldSchema
	Attrs      FieldSchema
}

func (r ResourceSchema) ToProto() *providerpb.ResourceSchema {
	return &providerpb.ResourceSchema{
		Type:       r.Type,
		Identifier: r.Identifier.ToProto(),
		Config:     r.Config.ToProto(),
		Attrs:      r.Attrs.ToProto(),
	}
}

type FieldSchema struct {
	Name         string
	Type         FieldType
	Fields       []FieldSchema
	IsIdentifier bool
}

func (f FieldSchema) ToProto() *providerpb.FieldSchema {
	fields := make([]*providerpb.FieldSchema, len(f.Fields))
	for i, field := range f.Fields {
		fields[i] = field.ToProto()
	}

	return &providerpb.FieldSchema{
		Name:         f.Name,
		Type:         f.Type.ToProto(),
		Fields:       fields,
		IsIdentifier: f.IsIdentifier,
	}
}

type FieldType string

func (f FieldType) ToProto() providerpb.FieldType {
	switch f {
	case FieldTypeEmpty:
		return providerpb.FieldType_EMPTY
	case FieldTypeString:
		return providerpb.FieldType_STRING
	case FieldTypeStruct:
		return providerpb.FieldType_STRUCT
	default:
		return providerpb.FieldType_EMPTY
	}
}

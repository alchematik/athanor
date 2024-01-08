package value

import (
	providerpb "github.com/alchematik/athanor/internal/gen/go/proto/provider/v1"
)

type Type interface {
	ToStateValue() StateValue
}

type IdentifierType interface {
	Type

	ResourceType() string
	Value() Type
}

func String(s string) StringType {
	return StringType(s)
}

type StringType string

func (s StringType) ToStateValue() StateValue {
	return StringStateValue(s)
}

func Bool(b bool) BoolType {
	return BoolType(b)
}

type BoolType bool

func (b BoolType) ToStateValue() StateValue {
	return BoolStateValue(b)
}

func Map(m map[string]Type) MapType {
	return MapType(m)
}

type MapType map[string]Type

func (m MapType) ToStateValue() StateValue {
	val := MapStateValue{}
	for k, v := range m {
		val[k] = v.ToStateValue()
	}

	return val
}

// func Identifier(t string, val Type) IdentifierType {
// 	return IdentifierType{
// 		ResourceType: t,
// 		Value:        val,
// 	}
// }

// type IdentifierType struct {
// 	ResourceType string
// 	Value        Type
// }

// func (id IdentifierType) ToStateValue() StateValue {
// 	return IdentifierStateValue{
// 		ResourceType: id.ResourceType,
// 		Value:        id.Value.ToStateValue(),
// 	}
// }

type Resource struct {
	Identifier IdentifierType
	Config     Type
	Attrs      Type
}

func (r Resource) ToStateValue() ResourceStateValue {
	return ResourceStateValue{
		Identifier: IdentifierStateValue{
			ResourceType: r.Identifier.ResourceType(),
			Value:        r.Identifier.Value().ToStateValue(),
		},
		Config: r.Config.ToStateValue(),
		Attrs:  r.Attrs.ToStateValue(),
	}
}

type StateValue interface {
	ToStateValueProto() *providerpb.Value
}

// type IdentifierStateValue interface {
// 	ToIdentifierStateValueProto() *providerpb.Identifier
// }

type BoolStateValue bool

func (b BoolStateValue) ToStateValueProto() *providerpb.Value {
	return &providerpb.Value{
		Type: &providerpb.Value_BoolValue{
			BoolValue: bool(b),
		},
	}
}

type StringStateValue string

func (s StringStateValue) ToStateValueProto() *providerpb.Value {
	return &providerpb.Value{
		Type: &providerpb.Value_StringValue{
			StringValue: string(s),
		},
	}
}

type MapStateValue map[string]StateValue

func (m MapStateValue) ToStateValueProto() *providerpb.Value {
	p := map[string]*providerpb.Value{}
	for k, v := range m {
		p[k] = v.ToStateValueProto()
	}

	return &providerpb.Value{
		Type: &providerpb.Value_Map{
			Map: &providerpb.MapValue{
				Entries: p,
			},
		},
	}
}

type IdentifierStateValue struct {
	ResourceType string
	Value        StateValue
}

func (id IdentifierStateValue) ToStateValueProto() *providerpb.Value {
	return &providerpb.Value{
		Type: &providerpb.Value_Identifier{
			Identifier: &providerpb.Identifier{
				Type:  id.ResourceType,
				Value: id.Value.ToStateValueProto(),
			},
		},
	}
}

type ResourceStateValue struct {
	Identifier IdentifierStateValue
	Config     StateValue
	Attrs      StateValue
}

func (r ResourceStateValue) ToStateValueProto() *providerpb.Resource {
	return &providerpb.Resource{
		Identifier: r.Identifier.ToStateValueProto().GetIdentifier(),
		Config:     r.Config.ToStateValueProto(),
		Attrs:      r.Attrs.ToStateValueProto(),
	}
}

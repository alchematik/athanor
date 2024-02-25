package state

type Type interface {
	isStateType()
}

type Environment struct {
	Type

	States map[string]Type
}

type Provider struct {
	Type

	Name    string
	Version string
}

type Resource struct {
	Type

	Provider   Provider
	Identifier Identifier
	Config     Type
	Attrs      Type
	Exists     Bool
}

type Identifier struct {
	Type

	Alias        string
	ResourceType string
	Value        Type
}

type File struct {
	Type

	Path     string
	Checksum string
}

type String struct {
	Type

	Value string
}

type Bool struct {
	Type

	Value bool
}

type Map struct {
	Type

	Entries map[string]Type
}

type List struct {
	Type

	Elements []Type
}

type Unknown struct {
	Type

	Name   string
	Object Type
}

type Nil struct {
	Type
}

type ResourceRef struct {
	Type

	Alias string
}

type Immutable struct {
	Type

	Value Type
}

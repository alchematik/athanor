package state

type Type interface {
	isStateType()
}

type Environment struct {
	Type

	Resources     map[string]Resource
	DependencyMap map[string][]string
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
}

type Identifier struct {
	Type

	ResourceType string
	Value        Type
}

type String struct {
	Type

	Value string
}

type Map struct {
	Type

	Entries map[string]Type
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

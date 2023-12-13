package state

type Type interface {
	isStateType()
}

type Environment struct {
	Type

	Objects       map[string]Type
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
	Identifier Type
	Config     Type

	Attrs Type
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
}

type Nil struct {
	Type
}

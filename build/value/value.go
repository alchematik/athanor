package value

type Type interface {
	isValueType()
}

type Provider struct {
	Type

	Name       string
	Version    string
	Dependants map[string]bool
}

type Resource struct {
	Type

	Provider   Provider
	Identifier Type
	Config     Type
	Attrs      Type
	Dependants map[string]bool
}

type String struct {
	Type

	Value string
}

type Map struct {
	Type

	Entries map[string]Type
}

type Unresolved struct {
	Type
}

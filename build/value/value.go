package value

type Type interface {
	isValueType()
}

type Provider struct {
	Type

	Identifier ProviderIdentifier
}

type Resource struct {
	Type

	Provider   Provider
	Identifier ResourceIdentifier
	Config     Type
	Attrs      Type
}

type ProviderIdentifier struct {
	Type

	Alias   string
	Name    string
	Version string
}

type ResourceIdentifier struct {
	Type

	Alias        string
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

type Unresolved struct {
	Type

	Name       string
	Object     Type
	Unresolved bool
}

type Nil struct {
	Type
}

type ResourceRef struct {
	Type

	Alias string
}

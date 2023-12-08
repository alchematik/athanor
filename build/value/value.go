package value

type Type interface {
	isValueType()
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
	Attrs      Type
	Children   map[string]bool
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

// type Unknown struct {
// 	Type
// }

type Nil struct {
	Type
}

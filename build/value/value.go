package value

type Type interface {
	isValueType()
}

type Resource struct {
	Type

	Name       string
	Identifier Type
	Config     Type
	Attrs      Type
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

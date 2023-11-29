package value

type Type interface {
	isValueType()
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

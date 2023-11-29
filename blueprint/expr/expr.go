package expr

type Type interface {
	isExprType() bool
}

type String struct {
	Type

	Value string
}

type Map struct {
	Type

	Entries map[string]Type
}

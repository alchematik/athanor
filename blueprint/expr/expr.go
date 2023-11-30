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

type Get struct {
	Type

	Name   string
	Object Type
}

type Nil struct {
	Type
}

type Provider struct {
	Type

	Name    Type
	Version Type
}

type Resource struct {
	Type

	Provider   Type
	Identifier Type
	Config     Type
}

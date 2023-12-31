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

type IOGet struct {
	Type

	Name   string
	Object Type
}

type GetProvider struct {
	Type

	Alias string
}

type GetResource struct {
	Type

	Alias string
}

type ProviderIdentifier struct {
	Type

	Alias   string
	Name    Type
	Version Type
}

type ResourceIdentifier struct {
	Type

	Alias        string
	ResourceType string
	Value        Type
}

type Nil struct {
	Type
}

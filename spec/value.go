package spec

type Value interface {
	isValueType()
}

type ValueBuild struct {
	Value

	Component []Component
}

type ValueBlueprint struct {
	Value

	Components []Component
}

type ValueProvider struct {
	Value

	Identifier ValueProviderIdentifier
}

type ValueResource struct {
	Value

	Provider   ValueProvider
	Identifier ValueResourceIdentifier
	Config     Value
	Attrs      Value
	Exists     Value
}

type ValueProviderIdentifier struct {
	Value

	Alias   string
	Name    string
	Version string
}

type ValueResourceIdentifier struct {
	Value

	Alias        string
	ResourceType string
	Literal      Value
}

type ValueString struct {
	Value

	Literal string
}

type ValueBool struct {
	Value

	Literal bool
}

type ValueMap struct {
	Value

	Entries map[string]Value
}

type ValueFile struct {
	Value

	Path string
}

type ValueUnresolved struct {
	Value

	Name       string
	Object     Value
	Unresolved bool
}

type ValueNil struct {
	Value
}

type ValueResourceRef struct {
	Value

	Alias string
}

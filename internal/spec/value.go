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

	Name    string
	Version string
}

type ValueResource struct {
	Value

	Provider   ValueProvider
	Identifier ValueResourceIdentifier
	Config     Value
	Attrs      Value
	Exists     Value
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

type ValueList struct {
	Value

	Elements []Value
}

type ValueFile struct {
	Value

	Path string
}

type ValueUnresolved struct {
	Value

	Name   string
	Object Value
}

type ValueNil struct {
	Value
}

type ValueResourceRef struct {
	Value

	Alias string
}

package spec

type Component interface {
	isComponentType()
}

type ComponentResource struct {
	Component

	Value ValueResource
}

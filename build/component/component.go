package component

type Type interface {
	isComponentType()
}

type Resource struct {
	Type
}

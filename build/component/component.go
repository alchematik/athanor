package component

import (
	"github.com/alchematik/athanor/build/value"
)

type Type interface {
	isComponentType()
}

type Resource struct {
	Type

	Value value.Resource
}

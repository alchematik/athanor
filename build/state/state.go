package state

import (
	"github.com/alchematik/athanor/build/value"
)

type Type interface {
	isStateType()
}

type Resource struct {
	Type

	Identifier value.Type
	Config     value.Type
	Attrs      value.Type
}

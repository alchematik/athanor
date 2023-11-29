package stmt

import (
	"github.com/alchematik/athanor/blueprint/expr"
)

type Type interface {
	isStmtType()
}

type Resource struct {
	Type

	Name       string
	Identifier expr.Type
	Config     expr.Type
}

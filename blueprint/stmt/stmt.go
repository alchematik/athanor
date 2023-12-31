package stmt

import (
	"github.com/alchematik/athanor/blueprint/expr"
)

type Type interface {
	isStmtType()
}

type Provider struct {
	Type

	Identifier expr.Type
}

type Resource struct {
	Type

	Identifier expr.Type
	Provider   expr.Type
	Config     expr.Type
}

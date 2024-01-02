package stmt

import (
	"github.com/alchematik/athanor/blueprint/expr"
)

type Type interface {
	isStmtType()
}

type Provider struct {
	Type

	Expr expr.Type
}

type Resource struct {
	Type

	Expr expr.Type
}

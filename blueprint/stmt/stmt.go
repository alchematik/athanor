package stmt

import (
	"github.com/alchematik/athanor/blueprint/expr"
)

type Type interface {
	isStmtType()
}

type Declare struct {
	Type

	Alias string
	Value expr.Type
}

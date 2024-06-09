package ast

import (
	"github.com/alchematik/athanor/internal/state"
)

type StmtBuild struct {
	ID           string
	Name         string
	RuntimeInput Expr[map[string]any]
	Stmts        []any
}

type StmtResource struct {
	ID       string
	Name     string
	Resource Expr[state.Resource]
}

type StmtWatcher struct {
	ID    string
	Name  string
	Value any
}

type Expr[T any] interface {
	Eval(*state.State) (T, error)
}

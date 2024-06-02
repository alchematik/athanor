package ast

import (
	"github.com/alchematik/athanor/internal/state"
)

type Stmt interface {
	Eval(*state.State) error
}

type StmtBuild struct {
	Name  string
	Build Build
}

func (s StmtBuild) Eval(*state.State) error {
	return nil
}

type StmtResource struct {
	Name     string
	Resource Expr[state.Resource]
}

func (s StmtResource) Eval(*state.State) error {
	return nil
}

type StmtWatcher struct {
	Name  string
	Value any
}

type Expr[T any] interface {
	Eval(*state.State) (T, error)
}

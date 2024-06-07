package ast

import (
	"github.com/alchematik/athanor/internal/state"
)

type Stmt interface {
	Eval(string, *state.State) error
}

type StmtBuild struct {
	Name  string
	Build Build
}

func (stmt StmtBuild) Eval(string, *state.State) error {
	return nil
}

type StmtResource struct {
	Name     string
	Resource Expr[state.Resource]
}

func (stmt StmtResource) Eval(id string, s *state.State) error {
	r, err := stmt.Resource.Eval(s)
	if err != nil {
		return err
	}

	s.Resources[id] = r
	return nil
}

type StmtWatcher struct {
	Name  string
	Value any
}

type Expr[T any] interface {
	Eval(*state.State) (T, error)
}

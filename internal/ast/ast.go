package ast

import (
	"context"

	"github.com/alchematik/athanor/internal/state"
)

type StmtBuild struct {
	ID           string
	Name         string
	BuildID      string
	Exists       Expr[bool]
	RuntimeInput Expr[map[state.Maybe[string]]state.Maybe[any]]
	Stmts        []any
}

type StmtResource struct {
	ID       string
	Name     string
	BuildID  string
	Exists   Expr[bool]
	Resource Expr[state.Resource]
}

type StmtWatcher struct {
	ID    string
	Name  string
	Value any
}

type Expr[T any] interface {
	Eval(context.Context, API, *state.State) (state.Maybe[T], error)
}

type API interface {
	EvalResource(context.Context, *state.Resource) error
}

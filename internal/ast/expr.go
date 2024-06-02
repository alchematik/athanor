package ast

import (
	"github.com/alchematik/athanor/internal/state"
)

type Literal[T any] struct {
	Value T
}

func (l Literal[T]) Eval(_ *state.State) (T, error) {
	return l.Value, nil
}

type Map[T any] struct {
	Value map[Expr[string]]Expr[any]
}

func (m Map[T]) Eval(s *state.State) (T, error) {
	out := map[string]any{}
	var val T
	for k, v := range m.Value {
		outKey, err := k.Eval(s)
		if err != nil {
			return val, err
		}

		outVal, err := v.Eval(s)
		if err != nil {
			return val, err
		}

		out[outKey] = outVal
	}

	switch v := any(&val).(type) {
	case *any:
		*v = out
	case *map[string]any:
		*v = out
	}

	return val, nil
}

type ResourceExpr[T any | state.Resource] struct {
	Exists     Expr[bool]
	Identifier Expr[any]
	Config     Expr[any]
}

func (r ResourceExpr[T]) Eval(s *state.State) (T, error) {
	var out T

	e, err := r.Exists.Eval(s)
	if err != nil {
		return out, err
	}

	id, err := r.Identifier.Eval(s)
	if err != nil {
		return out, err
	}

	c, err := r.Config.Eval(s)
	if err != nil {
		return out, err
	}

	resource := state.Resource{
		Exists:     e,
		Identifier: id,
		Config:     c,
	}

	switch o := any(&out).(type) {
	case *state.Resource:
		*o = resource
	case *any:
		*o = resource
	}

	return out, nil
}

type MapCollection map[string]Expr[any]

func (m MapCollection) Eval(s *state.State) (map[string]any, error) {
	out := map[string]any{}
	for k, v := range m {
		o, err := v.Eval(s)
		if err != nil {
			return nil, err
		}

		out[k] = o
	}

	return out, nil
}

type GetResource struct {
	Name string
	From any
}

func (g GetResource) Eval(s *state.State) (state.Resource, error) {
	// TODO: Handle "from".
	// return scope.GetResource(g.Name)
	return state.Resource{}, nil
}

type Build struct {
	RuntimeInput Expr[map[string]any]
	Blueprint    Blueprint
}

func (b Build) Eval(scope *Scope) (Build, error) {
	return b, nil
}

type LocalFile struct {
	Path string
}

type Blueprint struct {
	Stmts []Stmt
}

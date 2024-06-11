package ast

import (
	"github.com/alchematik/athanor/internal/state"
)

type Any[T any] struct {
	Value Expr[T]
}

func (a Any[T]) Eval(s *state.State) (any, error) {
	out, err := a.Value.Eval(s)
	if err != nil {
		return nil, err
	}

	return out, nil
}

type Literal[T any] struct {
	Value T
}

func (l Literal[T]) Eval(_ *state.State) (T, error) {
	return l.Value, nil
}

type Map[T any] struct {
	Value map[Expr[string]]Expr[T]
}

func (m Map[T]) Eval(s *state.State) (map[string]T, error) {
	out := map[string]T{}
	for k, v := range m.Value {
		outKey, err := k.Eval(s)
		if err != nil {
			return nil, err
		}

		outVal, err := v.Eval(s)
		if err != nil {
			return nil, err
		}

		out[outKey] = outVal
	}

	return out, nil
}

type ResourceExpr[T any | state.Resource] struct {
	Name       string
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
		Name:       r.Name,
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
	return state.Resource{
		Name: g.Name,
	}, nil
}

type LocalFile struct {
	Path string
}

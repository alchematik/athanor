package ast

import (
	"context"

	"github.com/alchematik/athanor/internal/state"
)

type Any[T any] struct {
	Value Expr[T]
}

func (a Any[T]) Eval(ctx context.Context, api API, s *state.State) (any, error) {
	out, err := a.Value.Eval(ctx, api, s)
	if err != nil {
		return nil, err
	}

	return out, nil
}

type Literal[T any] struct {
	Value T
}

func (l Literal[T]) Eval(_ context.Context, _ API, _ *state.State) (T, error) {
	return l.Value, nil
}

type Map[T any] struct {
	Value map[Expr[string]]Expr[T]
}

func (m Map[T]) Eval(ctx context.Context, api API, s *state.State) (map[string]T, error) {
	out := map[string]T{}
	for k, v := range m.Value {
		outKey, err := k.Eval(ctx, api, s)
		if err != nil {
			return nil, err
		}

		outVal, err := v.Eval(ctx, api, s)
		if err != nil {
			return nil, err
		}

		out[outKey] = outVal
	}

	return out, nil
}

type ResourceExpr struct {
	Name       string
	Exists     Expr[bool]
	Identifier Expr[any]
	Config     Expr[any]
}

func (r ResourceExpr) Eval(ctx context.Context, api API, s *state.State) (state.Resource, error) {
	e, err := r.Exists.Eval(ctx, api, s)
	if err != nil {
		return state.Resource{}, err
	}

	id, err := r.Identifier.Eval(ctx, api, s)
	if err != nil {
		return state.Resource{}, err
	}

	c, err := r.Config.Eval(ctx, api, s)
	if err != nil {
		return state.Resource{}, err
	}

	res := state.Resource{
		Name:       r.Name,
		Exists:     e,
		Identifier: id,
		Config:     c,
	}

	if err := api.EvalResource(ctx, &res); err != nil {
		return state.Resource{}, err
	}

	return res, nil
}

type MapCollection map[string]Expr[any]

func (m MapCollection) Eval(ctx context.Context, api API, s *state.State) (map[string]any, error) {
	out := map[string]any{}
	for k, v := range m {
		o, err := v.Eval(ctx, api, s)
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

func (g GetResource) Eval(_ context.Context, _ API, s *state.State) (state.Resource, error) {
	// TODO: Handle "from".
	return state.Resource{
		Name: g.Name,
	}, nil
}

type LocalFile struct {
	Path string
}

package ast

import (
	"context"
	"errors"

	"github.com/alchematik/athanor/internal/state"
)

type Any[T any] struct {
	Value Expr[T]
}

func (a Any[T]) Eval(ctx context.Context, api API, s *state.State) (state.Maybe[any], error) {
	out, err := a.Value.Eval(ctx, api, s)
	if err != nil {
		return state.Maybe[any]{}, err
	}

	return state.Maybe[any]{Value: out}, nil
}

type Literal[T any] struct {
	Value T
}

func (l Literal[T]) Eval(_ context.Context, _ API, _ *state.State) (state.Maybe[T], error) {
	return state.Maybe[T]{Value: l.Value}, nil
}

type Map[T any] struct {
	Value map[Expr[string]]Expr[T]
}

func (m Map[T]) Eval(ctx context.Context, api API, s *state.State) (state.Maybe[map[state.Maybe[string]]state.Maybe[T]], error) {
	out := state.Maybe[map[state.Maybe[string]]state.Maybe[T]]{
		Value: map[state.Maybe[string]]state.Maybe[T]{},
	}
	for k, v := range m.Value {
		outKey, err := k.Eval(ctx, api, s)
		if err != nil {
			return out, err
		}

		outVal, err := v.Eval(ctx, api, s)
		if err != nil {
			return out, err
		}

		out.Value[outKey] = outVal
	}

	return out, nil
}

type ResourceExpr struct {
	Name       string
	Identifier Expr[any]
	Config     Expr[any]
}

func (r ResourceExpr) Eval(ctx context.Context, api API, s *state.State) (state.Maybe[state.Resource], error) {
	id, err := r.Identifier.Eval(ctx, api, s)
	if err != nil {
		return state.Maybe[state.Resource]{}, err
	}

	c, err := r.Config.Eval(ctx, api, s)
	if err != nil {
		return state.Maybe[state.Resource]{}, err
	}

	res := state.Resource{
		Identifier: id,
		Config:     c,
	}

	if err := api.EvalResource(ctx, &res); err != nil {
		return state.Maybe[state.Resource]{}, err
	}

	return state.Maybe[state.Resource]{Value: res}, nil
}

type GetResource struct {
	Name string
	From any
}

func (g GetResource) Eval(_ context.Context, _ API, s *state.State) (state.Maybe[state.Resource], error) {
	// TODO: Handle implement.
	return state.Maybe[state.Resource]{}, errors.New("not implement")
}

type LocalFile struct {
	Path string
}

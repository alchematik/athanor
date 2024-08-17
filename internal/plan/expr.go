package plan

import (
	"context"
)

type StmtBuild struct {
	ID           string
	Name         string
	BuildID      string
	Exists       Expr[bool]
	RuntimeInput Expr[map[Maybe[string]]Maybe[any]]
	Stmts        []any
}

type StmtResource struct {
	ID      string
	Name    string
	BuildID string

	Exists     Expr[bool]
	Type       Expr[string]
	Provider   Expr[Provider]
	Identifier Expr[any]
	Config     Expr[any]
}

type Expr[T any] interface {
	Eval(context.Context, *Plan) (Maybe[T], error)
}

type Maybe[T any] struct {
	Value   T
	Unknown bool
}

func (m Maybe[T]) Unwrap() (T, bool) {
	return m.Value, !m.Unknown
}

func ToMaybeType[V any](m Maybe[any]) Maybe[V] {
	unwrapped, _ := m.Unwrap()
	v := unwrapped.(V)

	return Maybe[V]{Value: v, Unknown: m.Unknown}
}

type ExprAny[T any] struct {
	Value Expr[T]
}

func (e ExprAny[T]) Eval(ctx context.Context, p *Plan) (Maybe[any], error) {
	out, err := e.Value.Eval(ctx, p)
	if err != nil {
		return Maybe[any]{}, err
	}

	val, ok := out.Unwrap()
	return Maybe[any]{Value: val, Unknown: !ok}, nil
}

type ExprLiteral[T any] struct {
	Value T
}

func (e ExprLiteral[T]) Eval(_ context.Context, _ *Plan) (Maybe[T], error) {
	return Maybe[T]{Value: e.Value}, nil
}

type ExprMap map[Expr[string]]Expr[any]

func (e ExprMap) Eval(ctx context.Context, p *Plan) (Maybe[map[Maybe[string]]Maybe[any]], error) {
	out := map[Maybe[string]]Maybe[any]{}
	for k, v := range e {
		outKey, err := k.Eval(ctx, p)
		if err != nil {
			return Maybe[map[Maybe[string]]Maybe[any]]{}, err
		}

		outVal, err := v.Eval(ctx, p)
		if err != nil {
			return Maybe[map[Maybe[string]]Maybe[any]]{}, err
		}

		out[outKey] = outVal
	}

	return Maybe[map[Maybe[string]]Maybe[any]]{Value: out}, nil
}

type ExprProvider struct {
	Name    Expr[string]
	Version Expr[string]
}

func (e ExprProvider) Eval(ctx context.Context, p *Plan) (Maybe[Provider], error) {
	name, err := e.Name.Eval(ctx, p)
	if err != nil {
		return Maybe[Provider]{}, err
	}

	version, err := e.Version.Eval(ctx, p)
	if err != nil {
		return Maybe[Provider]{}, err
	}

	out := Provider{
		Name:    name,
		Version: version,
	}
	return Maybe[Provider]{Value: out}, nil
}

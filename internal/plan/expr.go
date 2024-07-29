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
	ID       string
	Name     string
	BuildID  string
	Exists   Expr[bool]
	Resource Expr[Resource]
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

func MaybeIsOfType[V any](m Maybe[any]) bool {
	v, _ := m.Unwrap()
	_, ok := v.(V)
	return ok
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

	return Maybe[any]{Value: out}, nil
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

type ExprResource struct {
	Name       string
	Provider   Expr[Provider]
	Type       Expr[string]
	Identifier Expr[any]
	Config     Expr[any]
}

func (e ExprResource) Eval(ctx context.Context, p *Plan) (Maybe[Resource], error) {
	id, err := e.Identifier.Eval(ctx, p)
	if err != nil {
		return Maybe[Resource]{}, err
	}

	c, err := e.Config.Eval(ctx, p)
	if err != nil {
		return Maybe[Resource]{}, err
	}

	t, err := e.Type.Eval(ctx, p)
	if err != nil {
		return Maybe[Resource]{}, err
	}

	provider, err := e.Provider.Eval(ctx, p)
	if err != nil {
		return Maybe[Resource]{}, err
	}

	res := Resource{
		Type:       t,
		Provider:   provider,
		Identifier: id,
		Config:     c,
		Attrs:      Maybe[any]{Unknown: true},
	}

	return Maybe[Resource]{Value: res}, nil
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

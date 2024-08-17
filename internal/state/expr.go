package state

import (
	"context"
)

type StmtBuild struct {
	ID           string
	Name         string
	BuildID      string
	Exists       Expr[bool]
	RuntimeInput Expr[map[string]any]
	Stmts        []any
}

type StmtResource struct {
	ID      string
	Name    string
	BuildID string

	Type       Expr[string]
	Provider   Expr[Provider]
	Identifier Expr[any]
}

type Expr[T any] interface {
	Eval(context.Context, *State) (T, error)
}

type Provider struct {
	Name    string
	Version string
}

type Resource struct {
	Type       string
	Provider   Provider
	Identifier any
	Config     any
	Attrs      any
}

type ExprAny[T any] struct {
	Value Expr[T]
}

func (e ExprAny[T]) Eval(ctx context.Context, s *State) (any, error) {
	out, err := e.Value.Eval(ctx, s)
	if err != nil {
		return nil, err
	}

	return out, nil
}

type ExprLiteral[T any] struct {
	Value T
}

func (e ExprLiteral[T]) Eval(_ context.Context, _ *State) (T, error) {
	return e.Value, nil
}

type ExprMap map[Expr[string]]Expr[any]

func (e ExprMap) Eval(ctx context.Context, s *State) (map[string]any, error) {
	m := map[string]any{}
	for k, v := range e {
		key, err := k.Eval(ctx, s)
		if err != nil {
			return nil, err
		}

		val, err := v.Eval(ctx, s)
		if err != nil {
			return nil, err
		}

		m[key] = val
	}

	return m, nil
}

type ExprProvider struct {
	Name    Expr[string]
	Version Expr[string]
}

func (e ExprProvider) Eval(ctx context.Context, s *State) (Provider, error) {
	name, err := e.Name.Eval(ctx, s)
	if err != nil {
		return Provider{}, err
	}

	version, err := e.Version.Eval(ctx, s)
	if err != nil {
		return Provider{}, err
	}

	return Provider{
		Name:    name,
		Version: version,
	}, nil
}

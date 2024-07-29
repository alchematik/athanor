package diff

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/internal/plan"
	"github.com/alchematik/athanor/internal/state"
)

type StmtBuild struct {
	ID                string
	Name              string
	BuildID           string
	Exists            Expr[DiffLiteral[bool]]
	Stmts             []any
	StateRuntimeInput state.Expr[map[string]any]
	PlanRuntimeInput  plan.Expr[map[plan.Maybe[string]]plan.Maybe[any]]
}

type StmtResource struct {
	ID       string
	Name     string
	BuildID  string
	Exists   Expr[DiffLiteral[bool]]
	Resource Expr[Resource]
}

type Type string

const (
	TypeCreate  Type = "create"
	TypeUpdate       = "update"
	TypeDelete       = "delete"
	TypeNoop         = "noop"
	TypeUnknown      = "unknown"
	TypeEmpty        = ""
)

type Expr[T any] interface {
	Eval(context.Context, *Diff) (DiffType[T], error)
}

type DiffType[T any] struct {
	Diff T
	Type Type
}

type ExprAny[T any] struct {
	Value Expr[T]
}

func (e ExprAny[T]) Eval(ctx context.Context, d *Diff) (DiffType[any], error) {
	res, err := e.Value.Eval(ctx, d)
	if err != nil {
		return DiffType[any]{}, err
	}

	return DiffType[any]{
		Diff: res.Diff,
		Type: res.Type,
	}, nil
}

type DiffLiteral[T any] struct {
	Plan  T
	State T
}

type ExprLiteral[T comparable] struct {
	Plan  plan.Expr[T]
	State state.Expr[T]
}

func (e ExprLiteral[T]) Eval(ctx context.Context, d *Diff) (DiffType[DiffLiteral[T]], error) {
	p, err := e.Plan.Eval(ctx, d.Plan)
	if err != nil {
		return DiffType[DiffLiteral[T]]{}, err
	}

	s, err := e.State.Eval(ctx, d.State)
	if err != nil {
		return DiffType[DiffLiteral[T]]{}, err
	}

	return diffLiteral[T](p, s)
}

type ExprMap struct {
	Plan  plan.ExprMap
	State state.ExprMap
}

type DiffMap map[string]DiffType[any]

func (e ExprMap) Eval(ctx context.Context, d *Diff) (DiffType[DiffMap], error) {
	s, err := e.State.Eval(ctx, d.State)
	if err != nil {
		return DiffType[DiffMap]{}, err
	}

	p, err := e.Plan.Eval(ctx, d.Plan)
	if err != nil {
		return DiffType[DiffMap]{}, err
	}

	return diffMap(p, s)
}

type ExprResource struct {
	Name  string
	Plan  plan.Expr[plan.Resource]
	State state.Expr[state.Resource]
}

func (e ExprResource) Eval(ctx context.Context, d *Diff) (DiffType[Resource], error) {
	// TODO: Handle not found.
	s, err := e.State.Eval(ctx, d.State)
	if err != nil {
		return DiffType[Resource]{}, err
	}

	p, err := e.Plan.Eval(ctx, d.Plan)
	if err != nil {
		return DiffType[Resource]{}, err
	}

	t := DiffType[Resource]{
		Diff: Resource{},
	}

	unwrapped, ok := p.Unwrap()
	if !ok {
		t.Type = TypeUnknown
		return t, nil
	}

	cfg, err := diffAny(unwrapped.Config, s.Config)
	if err != nil {
		return t, err
	}

	t.Diff.Config = cfg

	// TODO: Handle case where creating or deleting resource.
	t.Type = t.Diff.Config.Type

	return t, nil
}

func diffMap(p plan.Maybe[map[plan.Maybe[string]]plan.Maybe[any]], s map[string]any) (DiffType[DiffMap], error) {
	t := DiffType[DiffMap]{
		Diff: DiffMap{},
	}

	unwrapped, ok := p.Unwrap()
	if !ok {
		t.Type = TypeUnknown
		return t, nil
	}

	var isUnknown, isUpdate bool
	for k, v := range unwrapped {
		pk, ok := k.Unwrap()
		if !ok {
			isUnknown = true
			continue
		}

		sv := s[pk]

		var err error
		t.Diff[pk], err = diffAny(v, sv)
		if err != nil {
			return t, nil
		}

		if t.Diff[pk].Type == TypeUnknown {
			isUnknown = true
		}

		if t.Diff[pk].Type != TypeNoop {
			isUpdate = true
		}
	}

	switch {
	case isUnknown:
		t.Type = TypeUnknown
	case isUpdate:
		t.Type = TypeUpdate
	default:
		t.Type = TypeNoop
	}

	return t, nil
}

func diffAny(p plan.Maybe[any], s any) (DiffType[any], error) {
	switch {
	case plan.MaybeIsOfType[string](p):
		s, ok := s.(string)
		if !ok {
			return DiffType[any]{}, fmt.Errorf("cannot compare string and %T", s)
		}

		d, err := diffLiteral[string](plan.ToMaybeType[string](p), s)
		if err != nil {
			return DiffType[any]{}, err
		}

		return DiffType[any]{
			Diff: d.Diff,
			Type: d.Type,
		}, nil
	case plan.MaybeIsOfType[bool](p):
		s, ok := s.(bool)
		if !ok {
			return DiffType[any]{}, fmt.Errorf("cannot compare string and %T", s)
		}

		d, err := diffLiteral[bool](plan.ToMaybeType[bool](p), s)
		if err != nil {
			return DiffType[any]{}, err
		}

		return DiffType[any]{
			Diff: d.Diff,
			Type: d.Type,
		}, nil
	default:
		return DiffType[any]{}, nil
	}
}

func diffLiteral[T comparable](p plan.Maybe[T], s T) (DiffType[DiffLiteral[T]], error) {
	unwrapped, ok := p.Unwrap()

	t := DiffType[DiffLiteral[T]]{
		Diff: DiffLiteral[T]{
			Plan:  unwrapped,
			State: s,
		},
	}

	switch {
	case !ok:
		t.Type = TypeUnknown
	case unwrapped == s:
		t.Type = TypeNoop
	default:
		t.Type = TypeUpdate
	}

	return t, nil
}

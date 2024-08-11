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
	Exists            Expr[Literal[bool]]
	Stmts             []any
	StateRuntimeInput state.Expr[map[string]any]
	PlanRuntimeInput  plan.Expr[map[plan.Maybe[string]]plan.Maybe[any]]
}

type StmtResource struct {
	ID       string
	Name     string
	BuildID  string
	Exists   Expr[Literal[bool]]
	Resource Expr[Resource]
}

type Action string

const (
	ActionCreate  Action = "create"
	ActionUpdate         = "update"
	ActionDelete         = "delete"
	ActionNoop           = "noop"
	ActionUnknown        = "unknown"
	ActionEmpty          = ""
)

type Expr[T any] interface {
	Eval(context.Context, *DiffResult) (Diff[T], error)
}

type Diff[T any] struct {
	Diff   T
	Action Action
}

type ExprAny[T any] struct {
	Value Expr[T]
}

func (e ExprAny[T]) Eval(ctx context.Context, d *DiffResult) (Diff[any], error) {
	res, err := e.Value.Eval(ctx, d)
	if err != nil {
		return Diff[any]{}, err
	}

	return Diff[any]{
		Diff:   res.Diff,
		Action: res.Action,
	}, nil
}

type Literal[T any] struct {
	Plan  Emptyable[T]
	State Emptyable[T]
}

type ExprLiteral[T comparable] struct {
	Plan  plan.Expr[T]
	State state.Expr[T]
}

type Emptyable[T any] struct {
	Value   T
	IsEmpty bool
}

func (e ExprLiteral[T]) Eval(ctx context.Context, d *DiffResult) (Diff[Literal[T]], error) {
	p, err := e.Plan.Eval(ctx, d.Plan)
	if err != nil {
		return Diff[Literal[T]]{}, err
	}

	s, err := e.State.Eval(ctx, d.State)
	if err != nil {
		return Diff[Literal[T]]{}, err
	}

	return diffLiteral[T](
		Emptyable[plan.Maybe[T]]{Value: p},
		Emptyable[T]{Value: s},
	)
}

type ExprMap struct {
	Plan  plan.ExprMap
	State state.ExprMap
}

type Map map[Diff[Literal[string]]]Diff[any]

func (e ExprMap) Eval(ctx context.Context, d *DiffResult) (Diff[Map], error) {
	s, err := e.State.Eval(ctx, d.State)
	if err != nil {
		return Diff[Map]{}, err
	}

	p, err := e.Plan.Eval(ctx, d.Plan)
	if err != nil {
		return Diff[Map]{}, err
	}

	return diffMap(
		Emptyable[plan.Maybe[map[plan.Maybe[string]]plan.Maybe[any]]]{Value: p},
		Emptyable[map[string]any]{Value: s},
	)
}

type ExprResource struct {
	Name  string
	Plan  plan.Expr[plan.Resource]
	State state.Expr[state.Resource]
}

func (e ExprResource) Eval(ctx context.Context, d *DiffResult) (Diff[Resource], error) {
	// TODO: Handle not found.
	s, err := e.State.Eval(ctx, d.State)
	if err != nil {
		return Diff[Resource]{}, err
	}

	p, err := e.Plan.Eval(ctx, d.Plan)
	if err != nil {
		return Diff[Resource]{}, err
	}

	t := Diff[Resource]{
		Diff: Resource{},
	}

	unwrapped, ok := p.Unwrap()
	if !ok {
		t.Action = ActionUnknown
		return t, nil
	}

	// TODO: Handle not found.
	cfg, err := diffAny(
		Emptyable[plan.Maybe[any]]{Value: unwrapped.Config, IsEmpty: unwrapped.Config.Value == nil},
		Emptyable[any]{Value: s.Config, IsEmpty: s.Config == nil},
	)
	if err != nil {
		return t, err
	}

	t.Diff.Config = cfg
	t.Diff.Provider = Provider{
		Name:    s.Provider.Name,
		Version: s.Provider.Version,
	}

	// TODO: Handle case where creating or deleting resource.
	t.Action = t.Diff.Config.Action

	return t, nil
}

func diffMap(p Emptyable[plan.Maybe[map[plan.Maybe[string]]plan.Maybe[any]]], s Emptyable[map[string]any]) (Diff[Map], error) {
	switch {
	case p.IsEmpty && s.IsEmpty:
		return Diff[Map]{
			Action: ActionNoop,
			Diff:   Map{},
		}, nil
	case p.IsEmpty && !s.IsEmpty:
		m := Map{}
		for sk, sv := range s.Value {
			kd, err := diffLiteral[string](
				Emptyable[plan.Maybe[string]]{IsEmpty: true},
				Emptyable[string]{Value: sk},
			)
			if err != nil {
				return Diff[Map]{}, err
			}

			vd, err := diffAny(
				Emptyable[plan.Maybe[any]]{IsEmpty: true},
				Emptyable[any]{Value: sv},
			)
			if err != nil {
				return Diff[Map]{}, err
			}

			m[kd] = vd
		}

		return Diff[Map]{
			Action: ActionDelete,
			Diff:   m,
		}, nil
	case !p.IsEmpty && s.IsEmpty:
		m := Map{}
		unwrapped, ok := p.Value.Unwrap()
		if !ok {
			return Diff[Map]{
				Action: ActionUnknown,
				Diff:   m,
			}, nil
		}

		for k, v := range unwrapped {
			kd, err := diffLiteral[string](
				Emptyable[plan.Maybe[string]]{Value: k},
				Emptyable[string]{IsEmpty: true},
			)
			if err != nil {
				return Diff[Map]{}, err
			}

			vd, err := diffAny(
				Emptyable[plan.Maybe[any]]{Value: v},
				Emptyable[any]{IsEmpty: true},
			)
			if err != nil {
				return Diff[Map]{}, err
			}

			m[kd] = vd
		}

		return Diff[Map]{
			Action: ActionCreate,
			Diff:   m,
		}, nil
	default:
		m := Map{}
		unwrapped, ok := p.Value.Unwrap()
		if !ok {
			return Diff[Map]{
				Action: ActionUnknown,
				Diff:   m,
			}, nil
		}

		var isUpdate bool
		for k, v := range unwrapped {
			kp, ok := k.Unwrap()
			if !ok {
				kd := Diff[Literal[string]]{
					Action: ActionUnknown,
				}
				m[kd] = Diff[any]{
					Action: ActionUnknown,
				}
				continue
			}
			sv, present := s.Value[kp]
			if present {
				kd := Diff[Literal[string]]{
					Action: ActionNoop,
					Diff: Literal[string]{
						Plan:  Emptyable[string]{Value: kp},
						State: Emptyable[string]{Value: kp},
					},
				}

				vd, err := diffAny(
					Emptyable[plan.Maybe[any]]{Value: v},
					Emptyable[any]{Value: sv},
				)
				if err != nil {
					return Diff[Map]{}, err
				}

				isUpdate = isUpdate || vd.Action != ActionNoop

				m[kd] = vd
				continue
			}

			kd, err := diffLiteral[string](
				Emptyable[plan.Maybe[string]]{Value: k},
				Emptyable[string]{IsEmpty: true},
			)
			if err != nil {
				return Diff[Map]{}, err
			}

			vd, err := diffAny(
				Emptyable[plan.Maybe[any]]{Value: v},
				Emptyable[any]{IsEmpty: true},
			)
			if err != nil {
				return Diff[Map]{}, err
			}

			m[kd] = vd
		}

		for k, v := range s.Value {
			kd := Diff[Literal[string]]{
				Action: ActionNoop,
				Diff: Literal[string]{
					Plan:  Emptyable[string]{Value: k},
					State: Emptyable[string]{Value: k},
				},
			}

			_, ok := m[kd]
			if ok {
				// Diff already accounted for.
				continue
			}

			kd = Diff[Literal[string]]{
				Action: ActionDelete,
				Diff: Literal[string]{
					Plan:  Emptyable[string]{IsEmpty: true},
					State: Emptyable[string]{Value: k},
				},
			}

			vd, err := diffAny(
				Emptyable[plan.Maybe[any]]{IsEmpty: true},
				Emptyable[any]{Value: v},
			)
			if err != nil {
				return Diff[Map]{}, err
			}

			isUpdate = true
			m[kd] = vd
		}

		var action Action = ActionNoop
		if isUpdate {
			action = ActionUpdate
		}

		return Diff[Map]{
			Action: action,
			Diff:   m,
		}, nil
	}
}

func diffAny(p Emptyable[plan.Maybe[any]], s Emptyable[any]) (Diff[any], error) {
	if p.IsEmpty && s.IsEmpty {
		return Diff[any]{Action: ActionNoop}, nil
	}

	if p.IsEmpty {
		switch val := s.Value.(type) {
		case string:
			return Diff[any]{
				Action: ActionDelete,
				Diff: Literal[string]{
					Plan:  Emptyable[string]{IsEmpty: true},
					State: Emptyable[string]{Value: val},
				},
			}, nil
		case bool:
			return Diff[any]{
				Action: ActionDelete,
				Diff: Literal[bool]{
					Plan:  Emptyable[bool]{IsEmpty: true},
					State: Emptyable[bool]{Value: val},
				},
			}, nil
		case map[string]any:
			d, err := diffMap(
				Emptyable[plan.Maybe[map[plan.Maybe[string]]plan.Maybe[any]]]{IsEmpty: true},
				Emptyable[map[string]any]{Value: val},
			)
			if err != nil {
				return Diff[any]{}, err
			}

			return Diff[any]{
				Action: ActionDelete,
				Diff:   d,
			}, nil
		default:
			return Diff[any]{}, fmt.Errorf("unknown state diff type: %T", s.Value)
		}
	}

	switch planVal := p.Value.Value.(type) {
	case string:
		stringPlanVal, _ := p.Value.Value.(string)
		stringPlan := Emptyable[plan.Maybe[string]]{
			IsEmpty: p.IsEmpty,
			Value: plan.Maybe[string]{
				Unknown: p.Value.Unknown,
				Value:   stringPlanVal,
			},
		}
		stringStateVal, _ := s.Value.(string)
		stringState := Emptyable[string]{
			IsEmpty: s.IsEmpty,
			Value:   stringStateVal,
		}
		d, err := diffLiteral[string](stringPlan, stringState)
		if err != nil {
			return Diff[any]{}, err
		}

		return Diff[any]{
			Diff:   d.Diff,
			Action: d.Action,
		}, nil
	case bool:
		d, err := diffLiteral[bool](
			Emptyable[plan.Maybe[bool]]{
				IsEmpty: p.IsEmpty,
				Value: plan.Maybe[bool]{
					Unknown: p.Value.Unknown,
					Value:   p.Value.Value.(bool),
				},
			},
			Emptyable[bool]{
				IsEmpty: s.IsEmpty,
				Value:   s.Value.(bool),
			},
		)
		if err != nil {
			return Diff[any]{}, err
		}

		return Diff[any]{
			Diff:   d.Diff,
			Action: d.Action,
		}, nil
	case map[plan.Maybe[string]]plan.Maybe[any]:
		stateVal, _ := s.Value.(map[string]any)
		d, err := diffMap(
			Emptyable[plan.Maybe[map[plan.Maybe[string]]plan.Maybe[any]]]{
				IsEmpty: p.IsEmpty,
				Value: plan.Maybe[map[plan.Maybe[string]]plan.Maybe[any]]{
					Unknown: p.Value.Unknown,
					Value:   planVal,
				},
			},
			Emptyable[map[string]any]{
				IsEmpty: s.IsEmpty,
				Value:   stateVal,
			},
		)
		if err != nil {
			return Diff[any]{}, err
		}

		return Diff[any]{
			Diff:   d.Diff,
			Action: d.Action,
		}, nil
	default:
		return Diff[any]{}, fmt.Errorf("unknown diff type: %T", p.Value.Value)
	}
}

func diffLiteral[T comparable](p Emptyable[plan.Maybe[T]], s Emptyable[T]) (Diff[Literal[T]], error) {
	unwrapped, ok := p.Value.Unwrap()
	t := Diff[Literal[T]]{
		Diff: Literal[T]{
			Plan:  Emptyable[T]{Value: unwrapped},
			State: s,
		},
	}

	switch {
	case !ok:
		t.Action = ActionUnknown
	case p.IsEmpty && s.IsEmpty || unwrapped == s.Value:
		t.Action = ActionNoop
	case s.IsEmpty:
		t.Action = ActionCreate
	case p.IsEmpty:
		t.Action = ActionDelete
	default:
		t.Action = ActionUpdate
	}

	return t, nil
}

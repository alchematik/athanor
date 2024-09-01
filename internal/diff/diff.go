package diff

import (
	"fmt"
	"sync"

	"github.com/alchematik/athanor/internal/plan"
	"github.com/alchematik/athanor/internal/state"
)

type DiffResult struct {
	sync.Mutex

	Plan      *plan.Plan
	State     *state.State
	Resources map[string]*ResourceDiff
	Builds    map[string]*BuildDiff
}

func (d *DiffResult) Resource(id string) (*ResourceDiff, bool) {
	d.Lock()
	defer d.Unlock()

	r, ok := d.Resources[id]
	return r, ok
}

func (d *DiffResult) Build(id string) (*BuildDiff, bool) {
	d.Lock()
	defer d.Unlock()

	b, ok := d.Builds[id]
	return b, ok
}

type EvalState struct {
	State string
	Error error
}

type BuildDiff struct {
	sync.Mutex

	action    Action
	name      string
	evalState EvalState
	exists    Diff[Literal[bool]]
}

func (b *BuildDiff) ToDone() {
	b.Lock()
	defer b.Unlock()

	b.evalState.State = "done"
}

func (b *BuildDiff) ToEvaluating() {
	b.Lock()
	defer b.Unlock()

	b.evalState.State = "evaluating"
}

func (b *BuildDiff) GetEvalState() EvalState {
	b.Lock()
	defer b.Unlock()

	return b.evalState
}

func (b *BuildDiff) GetAction() Action {
	b.Lock()
	defer b.Unlock()

	return b.action
}

func (b *BuildDiff) GetName() string {
	b.Lock()
	defer b.Unlock()

	return b.name
}

func (b *BuildDiff) SetExists(exists Diff[Literal[bool]]) {
	b.Lock()
	defer b.Unlock()

	b.exists = exists
}

type ResourceDiff struct {
	sync.Mutex

	name      string
	evalState EvalState
	action    Action

	identifier   any
	provider     state.Provider
	resourceType string

	exists Diff[Literal[bool]]
	config Diff[any]
}

func (r *ResourceDiff) SetExists(exists Diff[Literal[bool]]) {
	r.Lock()
	defer r.Unlock()

	r.exists = exists
}

func (r *ResourceDiff) SetConfig(config Diff[any]) {
	r.Lock()
	defer r.Unlock()

	r.config = config
}

func (r *ResourceDiff) ToEvaluating() {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "evaluating"
}

func (r *ResourceDiff) ToDone() {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "done"
}

func (r *ResourceDiff) ToError(err error) {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "error"
	r.evalState.Error = err
}

func (r *ResourceDiff) GetEvalState() EvalState {
	r.Lock()
	defer r.Unlock()

	return r.evalState
}

func (r *ResourceDiff) GetName() string {
	r.Lock()
	defer r.Unlock()

	return r.name
}

func (r *ResourceDiff) GetProvider() state.Provider {
	r.Lock()
	defer r.Unlock()

	return r.provider
}

func (r *ResourceDiff) SetProvider(p state.Provider) {
	r.Lock()
	defer r.Unlock()

	r.provider = p
}

func (r *ResourceDiff) GetConfig() Diff[any] {
	r.Lock()
	defer r.Unlock()

	return r.config
}

func (r *ResourceDiff) SetIdentifier(id any) {
	r.Lock()
	defer r.Unlock()

	r.identifier = id
}

func (r *ResourceDiff) Identifier() any {
	r.Lock()
	defer r.Unlock()

	return r.identifier
}

func (r *ResourceDiff) SetAction(a Action) {
	r.Lock()
	defer r.Unlock()

	r.action = a
}

func (r *ResourceDiff) Action() Action {
	r.Lock()
	defer r.Unlock()

	return r.action
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

type Diff[T any] struct {
	Diff   T
	Action Action
}

type Literal[T any] struct {
	Plan  Emptyable[T]
	State Emptyable[T]
}

type Emptyable[T any] struct {
	Value   T
	IsEmpty bool
}

type Map map[Diff[Literal[string]]]Diff[any]

func DiffMap(p Emptyable[plan.Maybe[map[plan.Maybe[string]]plan.Maybe[any]]], s Emptyable[map[string]any]) (Diff[Map], error) {
	switch {
	case p.IsEmpty && s.IsEmpty:
		return Diff[Map]{
			Action: ActionNoop,
			Diff:   Map{},
		}, nil
	case p.IsEmpty && !s.IsEmpty:
		m := Map{}
		for sk, sv := range s.Value {
			kd, err := DiffLiteral[string](
				Emptyable[plan.Maybe[string]]{IsEmpty: true},
				Emptyable[string]{Value: sk},
			)
			if err != nil {
				return Diff[Map]{}, err
			}

			vd, err := DiffAny(
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
			kd, err := DiffLiteral[string](
				Emptyable[plan.Maybe[string]]{Value: k},
				Emptyable[string]{IsEmpty: true},
			)
			if err != nil {
				return Diff[Map]{}, err
			}

			vd, err := DiffAny(
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

				vd, err := DiffAny(
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

			kd, err := DiffLiteral[string](
				Emptyable[plan.Maybe[string]]{Value: k},
				Emptyable[string]{IsEmpty: true},
			)
			if err != nil {
				return Diff[Map]{}, err
			}

			vd, err := DiffAny(
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

			vd, err := DiffAny(
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

func DiffAny(p Emptyable[plan.Maybe[any]], s Emptyable[any]) (Diff[any], error) {
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
			d, err := DiffMap(
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
		d, err := DiffLiteral[string](stringPlan, stringState)
		if err != nil {
			return Diff[any]{}, err
		}

		return Diff[any]{
			Diff:   d.Diff,
			Action: d.Action,
		}, nil
	case bool:
		d, err := DiffLiteral[bool](
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
		d, err := DiffMap(
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

func DiffLiteral[T comparable](p Emptyable[plan.Maybe[T]], s Emptyable[T]) (Diff[Literal[T]], error) {
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

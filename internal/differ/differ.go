package differ

import (
	"context"
	"fmt"
	"sync"

	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/evaluator"
	"github.com/alchematik/athanor/internal/selector"
	"github.com/alchematik/athanor/internal/state"
)

type Differ struct {
	Target *evaluator.Evaluator
	Actual *evaluator.Evaluator
	Result diff.Environment

	Lock *sync.Mutex
}

func (d *Differ) Diff(ctx context.Context, s selector.Selector) (diff.Type, error) {
	target, err := d.Target.Eval(ctx, s)
	if err != nil {
		return nil, err
	}

	actual, err := d.Actual.Eval(ctx, s)
	if err != nil {
		return nil, err
	}

	e, ok := SelectDiffEnvironment(d.Result, s)
	if !ok {
		return nil, fmt.Errorf("cannot find environment with selector: %v", s)
	}

	var result diff.Type

	switch actual := actual.(type) {
	case state.Environment:
		current, ok := e.Diffs[s.Name].(diff.Environment)
		if !ok {
			d.Lock.Lock()
			e.Diffs[s.Name] = diff.Environment{
				Diffs: map[string]diff.Type{},
			}
			d.Lock.Unlock()

			return current, nil
		}

		ops := map[diff.Operation]int{}
		for _, v := range current.Diffs {
			ops[v.Operation()]++
		}

		var op diff.Operation
		switch {
		case ops[diff.OperationUpdate] > 0:
			op = diff.OperationUpdate
		case ops[diff.OperationUnknown] > 0:
			op = diff.OperationUnknown
		case len(current.Diffs) == ops[diff.OperationCreate]:
			op = diff.OperationCreate
		case len(current.Diffs) == ops[diff.OperationDelete]:
			op = diff.OperationDelete
		case len(current.Diffs) == ops[diff.OperationNoop]:
			op = diff.OperationNoop
		default:
			op = diff.OperationUpdate
		}

		current.Dependencies = actual.DependencyMap
		current.DiffOperation = op
		d.Lock.Lock()
		e.Diffs[s.Name] = current
		d.Lock.Unlock()
		return current, nil
	case state.Resource:
		if target == nil {
			result, err = diff.Diff(actual, state.Resource{})
		} else {
			target, ok := target.(state.Resource)
			if !ok {
				return nil, fmt.Errorf("invalid diff: trying to compare Resource to %T", target)
			}

			result, err = diff.Diff(actual, target)
		}
		if err != nil {
			return nil, err
		}

		d.Lock.Lock()
		e.Diffs[s.Name] = result
		d.Lock.Unlock()
		return result, nil
	case nil:
		switch target := target.(type) {
		case state.Resource:
			result, err := diff.Diff(state.Resource{}, target)
			if err != nil {
				return nil, err
			}

			d.Lock.Lock()
			e.Diffs[s.Name] = result
			d.Lock.Unlock()
			return result, nil
		default:
			return nil, fmt.Errorf("unhandled target type for diff: %T", target)
		}
	default:
		return nil, fmt.Errorf("unhandled type for diff: %T", actual)
	}
}

func SelectDiffEnvironment(env diff.Environment, selector selector.Selector) (diff.Environment, bool) {
	if selector.Parent == nil {
		return env, true
	}

	parent, ok := SelectDiffEnvironment(env, *selector.Parent)
	if !ok {
		return diff.Environment{}, false
	}

	d, ok := parent.Diffs[selector.Parent.Name]
	if !ok {
		return diff.Environment{}, false
	}

	sub, ok := d.(diff.Environment)

	return sub, true
}

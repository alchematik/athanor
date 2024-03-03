package differ

import (
	"context"
	"fmt"
	"sync"

	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/state"
)

type Differ struct {
	sync.Mutex
}

func (d *Differ) Diff(ctx context.Context, e diff.Environment, alias string, target, actual state.Type) (diff.Type, error) {
	var result diff.Type

	switch actual := actual.(type) {
	case state.Environment:
		current, ok := e.Diffs[alias].(diff.Environment)
		if !ok {
			to, ok := target.(state.Environment)
			if !ok {
				return nil, fmt.Errorf("expected env diff, got %T", target)
			}
			d.Lock()
			e.Diffs[alias] = diff.Environment{
				Diffs: map[string]diff.Type{},
				From:  actual,
				To:    to,
			}
			d.Unlock()

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

		current.DiffOperation = op
		d.Lock()
		e.Diffs[alias] = current
		d.Unlock()
		return current, nil
	case state.Resource:
		var err error
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

		d.Lock()
		e.Diffs[alias] = result
		d.Unlock()
		return result, nil
	case nil:
		switch target := target.(type) {
		case state.Resource:
			result, err := diff.Diff(state.Resource{}, target)
			if err != nil {
				return nil, err
			}

			d.Lock()
			e.Diffs[alias] = result
			d.Unlock()
			return result, nil
		default:
			return nil, fmt.Errorf("unhandled target type for diff: %T", target)
		}
	default:
		return nil, fmt.Errorf("unhandled type for diff: %T", actual)
	}
}

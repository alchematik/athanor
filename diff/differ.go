package diff

import (
	"fmt"
	"sort"

	"github.com/alchematik/athanor/state"
)

type Differ struct {
}

type Diff struct {
	Name      string
	From      any
	To        any
	Operation Operation
	Diffs     []Diff
}

type Operation string

const (
	OperationEmpty  Operation = ""
	OperationNoop   Operation = "noop"
	OperationCreate Operation = "create"
	OperationUpdate Operation = "update"
	OperationDelete Operation = "delete"
)

func DiffTypes(from, to state.Type) (Diff, error) {
	switch f := from.(type) {
	case state.String:
		t, ok := to.(state.String)
		if !ok {
			return Diff{}, fmt.Errorf("expected type %T, got %T", f, to)
		}

		return String(f, t)
	case state.Map:
		t, ok := to.(state.Map)
		if !ok {
			return Diff{}, fmt.Errorf("expected type %T, got %T", f, to)
		}

		return Map(f, t)
	case state.Resource:
		t, ok := to.(state.Resource)
		if !ok {
			return Diff{}, fmt.Errorf("expected type %T, got %T", f, to)
		}

		return Resource(f, t)
	default:
		return Diff{}, fmt.Errorf("unsupported type: %T", from)
	}
}

// func Environment(from, to state.Environment) (Diff, error) {
//   for k, v := range
// }

func Resource(from, to state.Resource) (Diff, error) {
	config, err := DiffTypes(from.Config, to.Config)
	if err != nil {
		return Diff{}, err
	}

	config.Name = "config"

	return Diff{
		Operation: config.Operation,
		From:      from,
		To:        to,
		Diffs: []Diff{
			config,
		},
	}, nil
}

func String(from, to state.String) (Diff, error) {
	var op Operation

	switch {
	case to.Value == "" && from.Value != "":
		op = OperationDelete
	case to.Value != "" && from.Value == "":
		op = OperationCreate
	case to.Value == from.Value:
		op = OperationNoop
	default:
		op = OperationUpdate
	}

	d := Diff{
		From:      from,
		To:        to,
		Operation: op,
	}
	return d, nil
}

func Map(from, to state.Map) (Diff, error) {
	var op Operation
	var diffs []Diff

	switch {
	case len(to.Entries) == 0 && len(from.Entries) != 0:
		op = OperationDelete
		for k, v := range from.Entries {
			diffs = append(diffs, Diff{
				Operation: OperationDelete,
				Name:      k,
				To:        state.Nil{},
				From:      v,
			})
		}
	case len(to.Entries) != 0 && len(from.Entries) == 0:
		op = OperationCreate
		for k, v := range to.Entries {
			diffs = append(diffs, Diff{
				Operation: OperationCreate,
				Name:      k,
				To:        v,
				From:      state.Nil{},
			})
		}
	default:
		for k, v := range to.Entries {
			fromVal, ok := from.Entries[k]
			if !ok {
				diffs = append(diffs, Diff{
					Name:      k,
					To:        v,
					From:      state.Nil{},
					Operation: OperationCreate,
				})
				continue
			}

			diff, err := DiffTypes(fromVal, v)
			if err != nil {
				return Diff{}, err
			}

			diff.Name = k

			diffs = append(diffs, diff)
		}

		for k, v := range from.Entries {
			_, ok := to.Entries[k]
			if !ok {
				diffs = append(diffs, Diff{
					Name:      k,
					To:        state.Nil{},
					From:      v,
					Operation: OperationDelete,
				})
			}
		}
	}

	if op == OperationEmpty {
		op = OperationNoop

		for _, diff := range diffs {
			if diff.Operation != OperationNoop {
				op = OperationUpdate
				break
			}
		}
	}

	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Name < diffs[j].Name
	})

	d := Diff{
		From:      from,
		To:        to,
		Operation: op,
		Diffs:     diffs,
	}
	return d, nil
}

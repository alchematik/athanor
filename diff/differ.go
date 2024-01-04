package diff

import (
	"fmt"

	"github.com/alchematik/athanor/state"
)

type Type interface {
	Operation() Operation
}

type String struct {
	From          state.String
	To            state.String
	DiffOperation Operation
}

func (s String) Operation() Operation {
	return s.DiffOperation
}

type Bool struct {
	From          state.Bool
	To            state.Bool
	DiffOperation Operation
}

func (b Bool) Operation() Operation {
	return b.DiffOperation
}

type Map struct {
	From          state.Map
	To            state.Map
	Diffs         map[string]Type
	DiffOperation Operation
}

func (m Map) Operation() Operation {
	return m.DiffOperation
}

type Resource struct {
	From          state.Resource
	To            state.Resource
	ConfigDiff    Type
	ExistsDiff    Type
	DiffOperation Operation
}

func (r Resource) Operation() Operation {
	return r.DiffOperation
}

type Environment struct {
	From          state.Environment
	To            state.Environment
	Diffs         map[string]Type
	Dependencies  map[string][]string
	DiffOperation Operation
}

func (e Environment) Operation() Operation {
	return e.DiffOperation
}

type Unknown struct {
}

func (u Unknown) Operation() Operation {
	return OperationUnknown
}

type Operation string

const (
	OperationEmpty   Operation = ""
	OperationNoop    Operation = "noop"
	OperationCreate  Operation = "create"
	OperationUpdate  Operation = "update"
	OperationDelete  Operation = "delete"
	OperationUnknown Operation = "unknown"
)

func Diff(from, to state.Type) (Type, error) {
	_, fromIsUnknown := from.(state.Unknown)
	_, toIsUnknown := to.(state.Unknown)
	if fromIsUnknown || toIsUnknown {
		return Unknown{}, nil
	}

	_, fromIsEmpty := from.(state.Nil)
	if fromIsEmpty {
		switch t := to.(type) {
		case state.String:
			return stringDiff(state.String{}, t)
		case state.Bool:
			return boolDiff(state.Bool{}, t)
		case state.Map:
			return mapDiff(state.Map{}, t)
		case state.Resource:
			return resourceDiff(state.Resource{}, t)
		default:
			return nil, fmt.Errorf("invalid type for nil diff: %T", to)
		}
	}

	_, toIsEmpty := to.(state.Nil)

	switch f := from.(type) {
	case state.Environment:
		if toIsEmpty {
			return environmentDiff(f, state.Environment{})
		}

		t, ok := to.(state.Environment)
		if !ok {
			return nil, fmt.Errorf("invalid type for environment diff: %T", to)
		}

		return environmentDiff(f, t)
	case state.String:
		if toIsEmpty {
			return stringDiff(f, state.String{})
		}

		t, ok := to.(state.String)
		if !ok {
			return nil, fmt.Errorf("invalid type for string diff: %T", to)
		}

		return stringDiff(f, t)
	case state.Bool:
		if toIsEmpty {
			return boolDiff(f, state.Bool{})
		}

		t, ok := to.(state.Bool)
		if !ok {
			return nil, fmt.Errorf("invalid type for bool diff: %T", to)
		}

		return boolDiff(f, t)
	case state.Map:
		if toIsEmpty {
			return mapDiff(f, state.Map{Entries: map[string]state.Type{}})
		}

		t, ok := to.(state.Map)
		if !ok {
			return nil, fmt.Errorf("invalid type for map diff: %T", to)
		}

		return mapDiff(f, t)
	case state.Resource:
		if toIsEmpty {
			return resourceDiff(f, state.Resource{})
		}

		t, ok := to.(state.Resource)
		if !ok {
			return nil, fmt.Errorf("invalid type for resource diff: %T", to)
		}

		return resourceDiff(f, t)
	default:
		return nil, fmt.Errorf("unsupported type for diff: %T", from)
	}
}

func environmentDiff(from, to state.Environment) (Environment, error) {
	var op Operation
	diffs := map[string]Type{}
	var depMap map[string][]string

	switch {
	case len(from.Resources) == 0 && len(to.Resources) > 0:
		op = OperationCreate
		depMap = to.DependencyMap
		for k, v := range to.Resources {
			d, err := Diff(state.Nil{}, v)
			if err != nil {
				return Environment{}, err
			}

			diffs[k] = d
		}
	case len(from.Resources) > 0 && len(to.Resources) == 0:
		op = OperationDelete

		for k, v := range to.Resources {
			d, err := Diff(v, state.Nil{})
			if err != nil {
				return Environment{}, err
			}

			diffs[k] = d
		}
	default:
		depMap = map[string][]string{}

		for k, v := range to.Resources {
			var fromVal state.Type
			fromVal, ok := from.Resources[k]
			if !ok {
				fromVal = state.Nil{}
			}

			d, err := Diff(fromVal, v)
			if err != nil {
				return Environment{}, err
			}

			switch d.Operation() {
			case OperationCreate, OperationUpdate, OperationUnknown:
				depMap[k] = to.DependencyMap[k]
			case OperationDelete:
				// This is meant to makes sure that children are deleted before parents.
				// TODO: Add tests around this behavior and also validate that child resources of the deleted parent
				// either don't depend on the parent or are also being deleted.
				children := from.DependencyMap[k]
				for _, child := range children {
					depMap[child] = append(depMap[child], k)
				}
			default:
				return Environment{}, fmt.Errorf("unsupported operation for environment resource: %v", d.Operation())
			}

			diffs[k] = d
		}

		if op == OperationEmpty {
			var hasUnknown bool
			var hasUpdate bool
			for _, d := range diffs {
				switch d.Operation() {
				case OperationUnknown:
					hasUnknown = true
				case OperationNoop:
				default:
					hasUpdate = true
				}
			}

			if hasUpdate {
				op = OperationUpdate
			} else if hasUnknown {
				op = OperationUnknown
			} else {
				op = OperationNoop
			}
		}
	}

	return Environment{
		From:          from,
		To:            to,
		DiffOperation: op,
		Diffs:         diffs,
		Dependencies:  depMap,
	}, nil
}

func resourceDiff(from, to state.Resource) (Resource, error) {
	config, err := Diff(from.Config, to.Config)
	if err != nil {
		return Resource{}, err
	}

	exists, err := Diff(from.Exists, to.Exists)
	if err != nil {
		return Resource{}, err
	}

	op := exists.Operation()
	if op == OperationNoop {
		op = config.Operation()
	}

	return Resource{
		DiffOperation: op,
		From:          from,
		To:            to,
		ConfigDiff:    config,
		ExistsDiff:    exists,
	}, nil
}

func stringDiff(from, to state.String) (String, error) {
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

	return String{
		From:          from,
		To:            to,
		DiffOperation: op,
	}, nil
}

func boolDiff(from, to state.Bool) (Bool, error) {
	var op Operation

	switch {
	case to.Value == from.Value:
		op = OperationNoop
	case to.Value && !from.Value:
		op = OperationCreate
	case !to.Value && from.Value:
		op = OperationDelete
	}

	return Bool{
		From:          from,
		To:            to,
		DiffOperation: op,
	}, nil
}

func mapDiff(from, to state.Map) (Map, error) {
	var op Operation
	diffs := map[string]Type{}

	switch {
	case len(from.Entries) != 0 && len(to.Entries) == 0:
		op = OperationDelete
		for k, v := range from.Entries {
			d, err := Diff(v, state.Nil{})
			if err != nil {
				return Map{}, err
			}

			diffs[k] = d
		}
	case len(from.Entries) == 0 && len(to.Entries) != 0:
		op = OperationCreate
		for k, v := range to.Entries {
			d, err := Diff(state.Nil{}, v)
			if err != nil {
				return Map{}, err
			}
			diffs[k] = d
		}
	default:
		for k, v := range to.Entries {
			fromVal, ok := from.Entries[k]
			if !ok {
				fromVal = state.Nil{}
			}

			d, err := Diff(fromVal, v)
			if err != nil {
				return Map{}, err
			}

			diffs[k] = d
		}

		for k, v := range from.Entries {
			_, ok := to.Entries[k]
			if !ok {
				d, err := Diff(v, state.Nil{})
				if err != nil {
					return Map{}, err
				}

				diffs[k] = d
			}
		}
	}

	if op == OperationEmpty {
		var hasUnknown bool
		var hasUpdate bool
		for _, d := range diffs {
			switch d.Operation() {
			case OperationUnknown:
				hasUnknown = true
			case OperationNoop:
			default:
				hasUpdate = true
			}
		}

		if hasUpdate {
			op = OperationUpdate
		} else if hasUnknown {
			op = OperationUnknown
		} else {
			op = OperationNoop
		}
	}

	return Map{
		From:          from,
		To:            to,
		Diffs:         diffs,
		DiffOperation: op,
	}, nil
}

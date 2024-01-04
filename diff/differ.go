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
	switch f := from.(type) {
	case state.Environment:
		switch t := to.(type) {
		case state.Environment:
			return EnvironmentDiff(f, t)
		case state.Nil:
			return EnvironmentDiff(f, state.Environment{})
		default:
			return nil, fmt.Errorf("invalid type for environment diff: %T", to)
		}
	case state.String:
		switch t := to.(type) {
		case state.String:
			return StringDiff(f, t)
		case state.Unknown:
			return Unknown{}, nil
		case state.Nil:
			return StringDiff(f, state.String{})
		default:
			return nil, fmt.Errorf("invalid type for string diff: %T", to)
		}
	case state.Map:
		switch t := to.(type) {
		case state.Map:
			return MapDiff(f, t)
		case state.Nil:
			return MapDiff(f, state.Map{Entries: map[string]state.Type{}})
		default:
			return nil, fmt.Errorf("invalid type for map diff: %T", to)
		}
	case state.Resource:
		switch t := to.(type) {
		case state.Resource:
			return ResourceDiff(f, t)
		case state.Nil:
			// TODO: Resource state.
			return ResourceDiff(f, state.Resource{})
		default:
			return nil, fmt.Errorf("invalid type for resource diff: %T", to)
		}
	case state.Nil:
		switch t := to.(type) {
		case state.String:
			return StringDiff(state.String{}, t)
		case state.Map:
			return MapDiff(state.Map{}, t)
		case state.Resource:
			return ResourceDiff(state.Resource{}, t)
		case state.Unknown:
			return Unknown{}, nil
		default:
			return nil, fmt.Errorf("invalid type for nil diff: %T", to)
		}
	case state.Unknown:
		return Unknown{}, nil
	default:
		return nil, fmt.Errorf("unsupported type for diff: %T", from)
	}
}

func EnvironmentDiff(from, to state.Environment) (Environment, error) {
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
		DiffOperation: op,
		Diffs:         diffs,
		Dependencies:  depMap,
	}, nil
}

func ResourceDiff(from, to state.Resource) (Resource, error) {
	config, err := Diff(from.Config, to.Config)
	if err != nil {
		return Resource{}, err
	}

	// TODO: take state into account i.e. exists vs not_exists
	// TODO: Take unknown fields into account.

	return Resource{
		DiffOperation: config.Operation(),
		From:          from,
		To:            to,
		ConfigDiff:    config,
	}, nil
}

func StringDiff(from, to state.String) (String, error) {
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

func MapDiff(from, to state.Map) (Map, error) {
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

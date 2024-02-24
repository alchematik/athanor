package diff

import (
	"fmt"
	"github.com/alchematik/athanor/internal/state"
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

type File struct {
	From          state.File
	To            state.File
	DiffOperation Operation
}

func (f File) Operation() Operation {
	return f.DiffOperation
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

type List struct {
	From          state.List
	To            state.List
	Diffs         []Type
	DiffOperation Operation
}

func (l List) Operation() Operation {
	return l.DiffOperation
}

type Identifier struct {
	From          state.Identifier
	To            state.Identifier
	DiffOperation Operation
}

func (d Identifier) Operation() Operation {
	return d.DiffOperation
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

type Immutable struct {
	From state.Type
	To   state.Type
}

func (i Immutable) Operation() Operation {
	return OperationNoop
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
		case state.File:
			return fileDiff(state.File{}, t)
		case state.List:
			return listDiff(state.List{}, t)
		case state.Identifier:
			return identifierDiff(state.Identifier{}, t)
		default:
			return nil, fmt.Errorf("invalid type for nil diff: %T, %+v", to, to)
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
	case state.Identifier:
		if toIsEmpty {
			return identifierDiff(f, state.Identifier{})
		}

		t, ok := to.(state.Identifier)
		if !ok {
			return nil, fmt.Errorf("invalid type for identifier diff: %T", to)
		}

		return identifierDiff(f, t)
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
	case state.List:
		if toIsEmpty {
			return listDiff(f, state.List{})
		}

		t, ok := to.(state.List)
		if !ok {
			return nil, fmt.Errorf("invalid type for list diff: %T", to)
		}

		return listDiff(f, t)
	case state.Resource:
		if toIsEmpty {
			return resourceDiff(f, state.Resource{})
		}

		t, ok := to.(state.Resource)
		if !ok {
			return nil, fmt.Errorf("invalid type for resource diff: %T", to)
		}

		return resourceDiff(f, t)
	case state.File:
		if toIsEmpty {
			return fileDiff(f, state.File{})
		}

		t, ok := to.(state.File)
		if !ok {
			return nil, fmt.Errorf("invalid type for file diff: %T", to)
		}

		return fileDiff(f, t)
	case state.Immutable:
		return Immutable{
			From: f.Value,
			To:   to,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported type for diff: %T", from)
	}
}

func environmentDiff(from, to state.Environment) (Environment, error) {
	var op Operation
	diffs := map[string]Type{}
	var depMap map[string][]string

	switch {
	case len(from.States) == 0 && len(to.States) > 0:
		op = OperationCreate
		depMap = to.DependencyMap
		for k, v := range to.States {
			d, err := Diff(state.Nil{}, v)
			if err != nil {
				return Environment{}, err
			}

			diffs[k] = d
		}
	case len(from.States) > 0 && len(to.States) == 0:
		op = OperationDelete

		for k, v := range to.States {
			d, err := Diff(v, state.Nil{})
			if err != nil {
				return Environment{}, err
			}

			diffs[k] = d
		}
	default:
		depMap = map[string][]string{}

		for k, v := range to.States {
			var fromVal state.Type
			fromVal, ok := from.States[k]
			if !ok {
				fromVal = state.Nil{}
			}

			d, err := Diff(fromVal, v)
			if err != nil {
				return Environment{}, err
			}

			switch d.Operation() {
			case OperationCreate, OperationUpdate, OperationUnknown, OperationNoop:
				depMap[k] = to.DependencyMap[k]
			case OperationDelete:
				// This is meant to makes sure that children are deleted before parents.
				// TODO: Add tests around this behavior and also validate that child resources of the deleted parent
				// either don't depend on the parent or are also being deleted.
				children := from.DependencyMap[k]
				for _, child := range children {
					depMap[child] = append(depMap[child], k)
				}
				depMap[k] = children
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
	if op == OperationNoop && to.Exists.Value {
		// Consider the config diff operation if the resource exists,
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

func fileDiff(from, to state.File) (File, error) {
	var op Operation
	switch {
	case to.Checksum == from.Checksum:
		op = OperationNoop
	case to.Checksum != "" && from.Checksum == "":
		op = OperationCreate
	case to.Checksum == "" && from.Checksum != "":
		op = OperationDelete
	case to.Checksum != from.Checksum:
		op = OperationUpdate
	}
	return File{
		From:          from,
		To:            to,
		DiffOperation: op,
	}, nil
}

func identifierDiff(from, to state.Identifier) (Identifier, error) {
	if from.ResourceType == "" && to.ResourceType != "" {
		return Identifier{
			From:          from,
			To:            to,
			DiffOperation: OperationCreate,
		}, nil
	}

	if from.ResourceType != "" && to.ResourceType == "" {
		return Identifier{
			DiffOperation: OperationDelete,
		}, nil
	}

	if from.ResourceType != to.ResourceType {
		return Identifier{
			From:          from,
			To:            to,
			DiffOperation: OperationUpdate,
		}, nil
	}

	valDiff, err := Diff(from.Value, to.Value)
	if err != nil {
		return Identifier{}, err
	}

	return Identifier{
		From:          from,
		To:            to,
		DiffOperation: valDiff.Operation(),
	}, nil
}

func listDiff(from, to state.List) (List, error) {
	op := OperationNoop
	diffs := []Type{}
	switch {
	case len(from.Elements) != 0 && len(to.Elements) == 0:
		op = OperationDelete
		for _, e := range from.Elements {
			d, err := Diff(e, state.Nil{})
			if err != nil {
				return List{}, err
			}

			diffs = append(diffs, d)
		}
	case len(from.Elements) == 0 && len(to.Elements) != 0:
		op = OperationCreate
		for _, e := range to.Elements {
			d, err := Diff(state.Nil{}, e)
			if err != nil {
				return List{}, err
			}

			diffs = append(diffs, d)
		}
	case len(from.Elements) > len(to.Elements):
		op = OperationUpdate
		i := 0
		for i < len(to.Elements) {
			d, err := Diff(from.Elements[i], to.Elements[i])
			if err != nil {
				return List{}, err
			}

			diffs = append(diffs, d)
			i++
		}
		for i < len(from.Elements) {
			d, err := Diff(from.Elements[i], state.Nil{})
			if err != nil {
				return List{}, err
			}

			diffs = append(diffs, d)
			i++
		}
	case len(from.Elements) < len(to.Elements):
		op = OperationUpdate

		i := 0
		for i < len(from.Elements) {
			d, err := Diff(from.Elements[i], to.Elements[i])
			if err != nil {
				return List{}, err
			}

			diffs = append(diffs, d)
			i++
		}
		for i < len(to.Elements) {
			d, err := Diff(state.Nil{}, to.Elements[i])
			if err != nil {
				return List{}, err
			}

			diffs = append(diffs, d)
			i++
		}
	default:
		for i := range from.Elements {
			d, err := Diff(from.Elements[i], to.Elements[i])
			if err != nil {
				return List{}, err
			}

			if d.Operation() != OperationNoop {
				op = OperationUpdate
			}

			diffs = append(diffs, d)
		}
	}

	return List{
		From:          from,
		To:            to,
		Diffs:         diffs,
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

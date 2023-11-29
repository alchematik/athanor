package interpreter

import (
	"fmt"

	"github.com/alchematik/athanor/blueprint"
	"github.com/alchematik/athanor/blueprint/expr"
	"github.com/alchematik/athanor/blueprint/stmt"
	"github.com/alchematik/athanor/build"
	"github.com/alchematik/athanor/build/value"
)

// Stmts and expressions

type Interpreter struct {
	Environment Environment
}

type Environment struct {
	Objects map[string]value.Type
}

func (in Interpreter) Interpret(env Environment, b blueprint.Blueprint) (build.Build, error) {
	var bld build.Build

	for _, st := range b.Stmts {
		switch s := st.(type) {
		case stmt.Resource:
			r, err := in.InterpretResourceStmt(env, s)
			if err != nil {
				return bld, err
			}

			env.Objects[r.Name] = r

			bld.States = append(bld.States, r)
		}
	}

	return bld, nil
}

func (in Interpreter) InterpretResourceStmt(env Environment, r stmt.Resource) (value.Resource, error) {
	id, err := in.InterpretExpr(env, r.Identifier)
	if err != nil {
		return value.Resource{}, err
	}

	config, err := in.InterpretExpr(env, r.Identifier)
	if err != nil {
		return value.Resource{}, err
	}

	return value.Resource{
		Name:       r.Name,
		Identifier: id,
		Config:     config,
		Attrs:      value.Unresolved{},
	}, nil

}

func (in Interpreter) InterpretExpr(env Environment, ex expr.Type) (value.Type, error) {
	switch e := ex.(type) {
	case expr.String:
		return value.String{Value: e.Value}, nil
	case expr.Map:
		m := value.Map{Entries: map[string]value.Type{}}
		for k, v := range e.Entries {
			var err error
			m.Entries[k], err = in.InterpretExpr(env, v)
			if err != nil {
				return nil, err
			}
		}

		return m, nil
	case expr.Get:
		name := e.Name
		var m map[string]value.Type

		// Nil means we're accessing properties on current environment.
		if _, ok := e.Object.(expr.Nil); ok {
			m = env.Objects
		} else {
			objVal, err := in.InterpretExpr(env, e.Object)
			if err != nil {
				return nil, err
			}

			switch obj := objVal.(type) {
			case value.Map:
				m = obj.Entries
			case value.Resource:
				m = map[string]value.Type{
					"identifier": obj.Identifier,
					"config":     obj.Config,
					"attrs":      obj.Attrs,
				}
			default:
				return nil, fmt.Errorf("cannot access property %q", name)
			}
		}

		val, ok := m[name]
		if !ok {
			return nil, fmt.Errorf("property %q not set", name)
		}

		return val, nil
	default:
		return nil, fmt.Errorf("unknown expr %T", ex)
	}
}

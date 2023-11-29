package interpreter

import (
	"fmt"

	"github.com/alchematik/athanor/blueprint"
	"github.com/alchematik/athanor/blueprint/expr"
	"github.com/alchematik/athanor/blueprint/stmt"
	"github.com/alchematik/athanor/build"
	"github.com/alchematik/athanor/build/state"
	"github.com/alchematik/athanor/build/value"
)

// Stmts and expressions

type Interpreter struct {
	Environment Environment
}

type Environment struct {
	Objects map[string]state.Type
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

			bld.States = append(bld.States, r)
		}
	}

	return bld, nil
}

func (in Interpreter) InterpretResourceStmt(env Environment, r stmt.Resource) (state.Resource, error) {
	id, err := in.InterpretExpr(env, r.Identifier)
	if err != nil {
		return state.Resource{}, err
	}

	config, err := in.InterpretExpr(env, r.Identifier)
	if err != nil {
		return state.Resource{}, err
	}

	return state.Resource{
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
	default:
		return nil, fmt.Errorf("unknown expr %T", ex)
	}
}

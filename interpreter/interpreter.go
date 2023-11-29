package interpreter

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/blueprint"
	"github.com/alchematik/athanor/blueprint/expr"
	"github.com/alchematik/athanor/blueprint/stmt"
	"github.com/alchematik/athanor/build"
	"github.com/alchematik/athanor/build/value"
)

// Stmts and expressions

type Interpreter struct {
	ResourcesAPI ResourcesAPI
}

type Environment struct {
	Objects map[string]value.Type
}

type ResourcesAPI interface {
	FetchResource(ctx context.Context, r value.Resource) (value.Resource, error)
}

type NilResourcesAPI struct {
}

func (api NilResourcesAPI) FetchResource(ctx context.Context, r value.Resource) (value.Resource, error) {
	r.Attrs = value.Unresolved{}
	return r, nil
}

func (in Interpreter) Interpret(ctx context.Context, env Environment, b blueprint.Blueprint) (build.Build, error) {
	var bld build.Build

	for _, st := range b.Stmts {
		switch s := st.(type) {
		case stmt.Resource:
			r, err := in.InterpretResourceStmt(ctx, env, s)
			if err != nil {
				return bld, err
			}

			env.Objects[r.Name] = r

			bld.States = append(bld.States, r)
		}
	}

	return bld, nil
}

func (in Interpreter) InterpretResourceStmt(ctx context.Context, env Environment, r stmt.Resource) (value.Resource, error) {
	id, err := in.InterpretExpr(env, r.Identifier)
	if err != nil {
		return value.Resource{}, err
	}

	config, err := in.InterpretExpr(env, r.Config)
	if err != nil {
		return value.Resource{}, err
	}

	input := value.Resource{
		Identifier: id,
		Config:     config,
	}

	out, err := in.ResourcesAPI.FetchResource(ctx, input)
	if err != nil {
		return value.Resource{}, err
	}

	return value.Resource{
		Name:       r.Name,
		Identifier: id,
		Config:     out.Config,
		Attrs:      out.Attrs,
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
			case value.Unresolved:
				return value.Unresolved{}, nil
			default:
				return nil, fmt.Errorf("cannot access property %q", e.Name)
			}
		}

		val, ok := m[e.Name]
		if !ok {
			return nil, fmt.Errorf("property %q not set", e.Name)
		}

		return val, nil
	default:
		return nil, fmt.Errorf("unknown expr %T", ex)
	}
}

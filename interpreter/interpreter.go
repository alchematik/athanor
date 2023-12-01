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
	r.Attrs = value.Unresolved{
		Name:   "",
		Object: r,
	}
	return r, nil
}

func (in Interpreter) Interpret(ctx context.Context, env Environment, b blueprint.Blueprint) (build.Build, error) {
	var bld build.Build

	for _, st := range b.Stmts {
		switch s := st.(type) {
		case stmt.Declare:
			v, err := in.InterpretExpr(ctx, env, s, s.Value)
			if err != nil {
				return bld, err
			}

			env.Objects[s.Alias] = v

			switch val := v.(type) {
			case value.Provider:
				bld.Values = append(bld.Values, val)
			case value.Resource:
				bld.Values = append(bld.Values, val)
			}
		default:
			return bld, fmt.Errorf("unknown stmt %T", st)
		}
	}

	return bld, nil
}

func (in Interpreter) InterpretExpr(ctx context.Context, env Environment, st stmt.Type, ex expr.Type) (value.Type, error) {
	switch e := ex.(type) {
	case expr.String:
		return value.String{Value: e.Value}, nil
	case expr.Map:
		m := value.Map{Entries: map[string]value.Type{}}
		for k, v := range e.Entries {
			var err error
			m.Entries[k], err = in.InterpretExpr(ctx, env, st, v)
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
			if s, ok := st.(stmt.Declare); ok {
				dependantAlias := s.Alias
				switch v := env.Objects[e.Name].(type) {
				case value.Provider:
					v.Dependants[dependantAlias] = true
				case value.Resource:
					v.Dependants[dependantAlias] = true
				}
			}
		} else {
			objVal, err := in.InterpretExpr(ctx, env, st, e.Object)
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
				return value.Unresolved{
					Name:   e.Name,
					Object: objVal,
				}, nil
			default:
				return nil, fmt.Errorf("cannot access property %q", e.Name)
			}
		}

		val, ok := m[e.Name]
		if !ok {
			return nil, fmt.Errorf("property %q not set", e.Name)
		}

		return val, nil
	case expr.Provider:
		return in.InterpretProviderExpr(ctx, env, st, e)
	case expr.Resource:
		return in.InterpretResourceExpr(ctx, env, st, e)
	default:
		return nil, fmt.Errorf("unknown expr %T", ex)
	}
}

func (in Interpreter) InterpretProviderExpr(ctx context.Context, env Environment, st stmt.Type, p expr.Provider) (value.Provider, error) {
	name, err := in.InterpretExpr(ctx, env, st, p.Name)
	if err != nil {
		return value.Provider{}, err
	}

	nameStr, ok := name.(value.String)
	if !ok {
		return value.Provider{}, fmt.Errorf("name must be a string")
	}

	version, err := in.InterpretExpr(ctx, env, st, p.Version)
	if err != nil {
		return value.Provider{}, err
	}

	versionStr, ok := version.(value.String)
	if !ok {
		return value.Provider{}, fmt.Errorf("version must be a string")
	}

	return value.Provider{
		Name:       nameStr.Value,
		Version:    versionStr.Value,
		Dependants: map[string]bool{},
	}, nil
}

func (in Interpreter) InterpretResourceExpr(ctx context.Context, env Environment, st stmt.Type, r expr.Resource) (value.Resource, error) {
	provider, err := in.InterpretExpr(ctx, env, st, r.Provider)
	if err != nil {
		return value.Resource{}, err
	}

	providerVal, ok := provider.(value.Provider)
	if !ok {
		return value.Resource{}, fmt.Errorf("must use a valid provider for resource")
	}

	id, err := in.InterpretExpr(ctx, env, st, r.Identifier)
	if err != nil {
		return value.Resource{}, err
	}

	config, err := in.InterpretExpr(ctx, env, st, r.Config)
	if err != nil {
		return value.Resource{}, err
	}

	var alias string
	if s, ok := st.(stmt.Declare); ok {
		alias = s.Alias
	}

	res := value.Resource{
		Provider:   providerVal,
		Identifier: id,
		Config:     config,
		Attrs: value.Unresolved{
			Name: "attrs",
			Object: value.Unresolved{
				Name:   alias,
				Object: value.Nil{},
			},
		},
		Dependants: map[string]bool{},
	}

	return res, nil
}

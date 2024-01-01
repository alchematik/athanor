package interpreter

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/blueprint/expr"
	"github.com/alchematik/athanor/build/value"
)

func (in Interpreter) Expr(ctx context.Context, env Environment, ex expr.Type) (value.Type, []string, error) {
	switch e := ex.(type) {
	case expr.String:
		return value.String{Value: e.Value}, nil, nil
	case expr.Map:
		return in.Map(ctx, env, e)
	case expr.ProviderIdentifier:
		return in.ProviderIdentifier(ctx, env, e)
	case expr.ResourceIdentifier:
		return in.ResourceIdentifier(ctx, env, e)
	case expr.IOGet:
		return in.IOGet(ctx, env, e)
	case expr.Get:
		return in.Get(ctx, env, e)
	case expr.GetProvider:
		p, ok := env.Providers[e.Alias]
		if !ok {
			return nil, nil, fmt.Errorf("provider with alias %q does not exist", e.Alias)
		}

		return p, nil, nil
	case expr.GetResource:
		r, ok := env.Resources[e.Alias]
		if !ok {
			return nil, nil, fmt.Errorf("resource with alias %q does not exist", e.Alias)
		}

		return r, nil, nil
	default:
		return nil, nil, fmt.Errorf("unknown expr %T", ex)
	}
}

func (in Interpreter) Map(ctx context.Context, env Environment, e expr.Map) (value.Map, []string, error) {
	m := value.Map{Entries: map[string]value.Type{}}
	var children []string
	for k, v := range e.Entries {
		var err error
		var valChildren []string

		m.Entries[k], valChildren, err = in.Expr(ctx, env, v)
		if err != nil {
			return value.Map{}, nil, err
		}

		children = append(children, valChildren...)
	}

	return m, children, nil
}

func (in Interpreter) ProviderIdentifier(ctx context.Context, env Environment, e expr.ProviderIdentifier) (value.ProviderIdentifier, []string, error) {
	name, nameChildren, err := in.Expr(ctx, env, e.Name)
	if err != nil {
		return value.ProviderIdentifier{}, nil, err
	}

	nameStr, ok := name.(value.String)
	if !ok {
		return value.ProviderIdentifier{}, nil, fmt.Errorf("provider name must be a string")
	}

	version, versionChildren, err := in.Expr(ctx, env, e.Version)
	if err != nil {
		return value.ProviderIdentifier{}, nil, err
	}

	versionStr, ok := version.(value.String)
	if !ok {
		return value.ProviderIdentifier{}, nil, fmt.Errorf("provider version must be a string")
	}

	children := append(nameChildren, versionChildren...)

	return value.ProviderIdentifier{
		Alias:   e.Alias,
		Name:    nameStr.Value,
		Version: versionStr.Value,
	}, children, nil
}

func (in Interpreter) ResourceIdentifier(ctx context.Context, env Environment, e expr.ResourceIdentifier) (value.ResourceIdentifier, []string, error) {
	val, children, err := in.Expr(ctx, env, e.Value)
	if err != nil {
		return value.ResourceIdentifier{}, nil, err
	}

	return value.ResourceIdentifier{
		Alias:        e.Alias,
		ResourceType: e.ResourceType,
		Value:        val,
	}, append(children, e.Alias), nil
}

func (in Interpreter) IOGet(ctx context.Context, env Environment, e expr.IOGet) (value.Unresolved, []string, error) {
	objVal, children, err := in.Expr(ctx, env, e.Object)
	if err != nil {
		return value.Unresolved{}, nil, err
	}

	unresolved, ok := objVal.(value.Unresolved)
	if !ok {
		return value.Unresolved{}, nil, fmt.Errorf("property %q does not belong to unresolved object; use get", e.Name)
	}

	return value.Unresolved{
		Name:   e.Name,
		Object: unresolved,
	}, children, nil
}

func (in Interpreter) Get(ctx context.Context, env Environment, e expr.Get) (value.Type, []string, error) {
	var m map[string]value.Type

	objVal, children, err := in.Expr(ctx, env, e.Object)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, fmt.Errorf("property %q belongs to an unresolved object; use io_get", e.Name)
	default:
		return nil, nil, fmt.Errorf("cannot access property %q", e.Name)
	}

	val, ok := m[e.Name]
	if !ok {
		return nil, nil, fmt.Errorf("property %q not set", e.Name)
	}

	return val, children, nil
}

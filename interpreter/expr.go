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
		return in.mapExpr(ctx, env, e)
	case expr.Provider:
		return in.provider(ctx, env, e)
	case expr.ProviderIdentifier:
		return in.providerIdentifierExpr(ctx, env, e)
	case expr.Resource:
		return in.resource(ctx, env, e)
	case expr.ResourceIdentifier:
		return in.resourceIdentifierExpr(ctx, env, e)
	case expr.IOGet:
		return in.ioGetExpr(ctx, env, e)
	case expr.Get:
		return in.getExpr(ctx, env, e)
	case expr.GetProvider:
		p, ok := env.Providers[e.Alias]
		if !ok {
			return value.Provider{}, nil, fmt.Errorf("provider with alias %q does not exist", e.Alias)
		}

		return p, nil, nil
	case expr.GetResource:
		r, ok := env.Resources[e.Alias]
		if !ok {
			return value.Resource{}, nil, fmt.Errorf("resource with alias %q does not exist", e.Alias)
		}

		return r, []string{e.Alias}, nil
	default:
		return nil, nil, fmt.Errorf("unknown expr %T", ex)
	}
}

func (in Interpreter) provider(ctx context.Context, env Environment, e expr.Provider) (value.Provider, []string, error) {
	val, children, err := in.Expr(ctx, env, e.Identifier)
	if err != nil {
		return value.Provider{}, nil, err
	}

	id, ok := val.(value.ProviderIdentifier)
	if !ok {
		return value.Provider{}, nil, fmt.Errorf("expected ProviderIdentifier type, got %T", val)
	}

	return value.Provider{
		Identifier: id,
	}, children, nil
}

func (in Interpreter) resource(ctx context.Context, env Environment, e expr.Resource) (value.Resource, []string, error) {
	providerValue, providerChildren, err := in.Expr(ctx, env, e.Provider)
	if err != nil {
		return value.Resource{}, nil, err
	}

	provider, ok := providerValue.(value.Provider)
	if !ok {
		return value.Resource{}, nil, fmt.Errorf("expected Provider type, got %T", providerValue)
	}

	idVal, idChildren, err := in.Expr(ctx, env, e.Identifier)
	if err != nil {
		return value.Resource{}, nil, err
	}

	id, ok := idVal.(value.ResourceIdentifier)
	if !ok {
		return value.Resource{}, nil, fmt.Errorf("expected ResourceIdentifier, got %T", idVal)
	}

	configVal, configChildren, err := in.Expr(ctx, env, e.Config)
	if err != nil {
		return value.Resource{}, nil, err
	}

	children := append(providerChildren, idChildren...)
	children = append(children, configChildren...)
	var out []string
	for _, child := range children {
		// Filter out alias to self.
		if child == id.Alias {
			continue
		}

		out = append(out, child)
	}

	return value.Resource{
		Provider:   provider,
		Identifier: id,
		Config:     configVal,
		Attrs: value.Unresolved{
			Name: "attrs",
			Object: value.ResourceRef{
				Alias: id.Alias,
			},
		},
	}, out, nil
}

func (in Interpreter) mapExpr(ctx context.Context, env Environment, e expr.Map) (value.Map, []string, error) {
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

func (in Interpreter) providerIdentifierExpr(ctx context.Context, env Environment, e expr.ProviderIdentifier) (value.ProviderIdentifier, []string, error) {
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

	if e.Alias == "" {
		return value.ProviderIdentifier{}, nil, fmt.Errorf("must provide alias")
	}

	return value.ProviderIdentifier{
		Alias:   e.Alias,
		Name:    nameStr.Value,
		Version: versionStr.Value,
	}, children, nil
}

func (in Interpreter) resourceIdentifierExpr(ctx context.Context, env Environment, e expr.ResourceIdentifier) (value.ResourceIdentifier, []string, error) {
	val, children, err := in.Expr(ctx, env, e.Value)
	if err != nil {
		return value.ResourceIdentifier{}, nil, err
	}

	if e.Alias == "" {
		return value.ResourceIdentifier{}, nil, fmt.Errorf("must provide alias")
	}

	if e.ResourceType == "" {
		return value.ResourceIdentifier{}, nil, fmt.Errorf("must provide resource type")
	}

	return value.ResourceIdentifier{
		Alias:        e.Alias,
		ResourceType: e.ResourceType,
		Value:        val,
	}, append(children, e.Alias), nil
}

func (in Interpreter) ioGetExpr(ctx context.Context, env Environment, e expr.IOGet) (value.Unresolved, []string, error) {
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

func (in Interpreter) getExpr(ctx context.Context, env Environment, e expr.Get) (value.Type, []string, error) {
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

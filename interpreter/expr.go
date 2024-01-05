package interpreter

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/spec"
)

func (in Interpreter) Expr(ctx context.Context, b spec.Spec, ex ast.Expr) (spec.Value, []string, error) {
	switch e := ex.(type) {
	case ast.ExprString:
		return spec.ValueString{Literal: e.Value}, nil, nil
	case ast.ExprBool:
		return spec.ValueBool{Literal: e.Value}, nil, nil
	case ast.ExprBlueprint:
		return in.blueprintExpr(ctx, b, e)
	case ast.ExprMap:
		return in.mapExpr(ctx, b, e)
	case ast.ExprProvider:
		return in.provider(ctx, b, e)
	case ast.ExprProviderIdentifier:
		return in.providerIdentifierExpr(ctx, b, e)
	case ast.ExprResource:
		return in.resource(ctx, b, e)
	case ast.ExprResourceIdentifier:
		return in.resourceIdentifierExpr(ctx, b, e)
	case ast.ExprIOGet:
		return in.ioGetExpr(ctx, b, e)
	case ast.ExprGet:
		return in.getExpr(ctx, b, e)
	case ast.ExprGetProvider:
		p, ok := b.Providers[e.Alias]
		if !ok {
			return spec.ValueProvider{}, nil, fmt.Errorf("provider with alias %q does not exist", e.Alias)
		}

		return p, nil, nil
	case ast.ExprGetResource:
		r, ok := b.Resources[e.Alias]
		if !ok {
			return spec.ValueResource{}, nil, fmt.Errorf("resource with alias %q does not exist", e.Alias)
		}

		return r, []string{e.Alias}, nil
	default:
		return nil, nil, fmt.Errorf("unknown expr %T", ex)
	}
}

func (in Interpreter) blueprintExpr(ctx context.Context, s spec.Spec, e ast.ExprBlueprint) (spec.Value, []string, error) {
	for _, stmt := range e.Stmts {
		if err := in.Stmt(ctx, s, stmt); err != nil {
			return nil, nil, err
		}
	}

	return nil, nil, nil
}

func (in Interpreter) provider(ctx context.Context, b spec.Spec, e ast.ExprProvider) (spec.ValueProvider, []string, error) {
	val, children, err := in.Expr(ctx, b, e.Identifier)
	if err != nil {
		return spec.ValueProvider{}, nil, err
	}

	id, ok := val.(spec.ValueProviderIdentifier)
	if !ok {
		return spec.ValueProvider{}, nil, fmt.Errorf("expected ProviderIdentifier type, got %T", val)
	}

	return spec.ValueProvider{
		Identifier: id,
	}, children, nil
}

func (in Interpreter) resource(ctx context.Context, b spec.Spec, e ast.ExprResource) (spec.ValueResource, []string, error) {
	providerValue, providerChildren, err := in.Expr(ctx, b, e.Provider)
	if err != nil {
		return spec.ValueResource{}, nil, err
	}

	provider, ok := providerValue.(spec.ValueProvider)
	if !ok {
		return spec.ValueResource{}, nil, fmt.Errorf("expected Provider type, got %T", providerValue)
	}

	idVal, idChildren, err := in.Expr(ctx, b, e.Identifier)
	if err != nil {
		return spec.ValueResource{}, nil, err
	}

	id, ok := idVal.(spec.ValueResourceIdentifier)
	if !ok {
		return spec.ValueResource{}, nil, fmt.Errorf("expected ResourceIdentifier, got %T", idVal)
	}

	configVal, configChildren, err := in.Expr(ctx, b, e.Config)
	if err != nil {
		return spec.ValueResource{}, nil, err
	}

	existsVal, existsChildren, err := in.Expr(ctx, b, e.Exists)
	if err != nil {
		return spec.ValueResource{}, nil, err
	}

	children := append(providerChildren, idChildren...)
	children = append(children, configChildren...)
	children = append(children, existsChildren...)
	var out []string
	for _, child := range children {
		// Filter out alias to self.
		if child == id.Alias {
			continue
		}

		out = append(out, child)
	}

	return spec.ValueResource{
		Provider:   provider,
		Identifier: id,
		Config:     configVal,
		Exists:     existsVal,
		Attrs: spec.ValueUnresolved{
			Name: "attrs",
			Object: spec.ValueResourceRef{
				Alias: id.Alias,
			},
		},
	}, out, nil
}

func (in Interpreter) mapExpr(ctx context.Context, b spec.Spec, e ast.ExprMap) (spec.ValueMap, []string, error) {
	m := spec.ValueMap{Entries: map[string]spec.Value{}}
	var children []string
	for k, v := range e.Entries {
		var err error
		var valChildren []string

		m.Entries[k], valChildren, err = in.Expr(ctx, b, v)
		if err != nil {
			return spec.ValueMap{}, nil, err
		}

		children = append(children, valChildren...)
	}

	return m, children, nil
}

func (in Interpreter) providerIdentifierExpr(ctx context.Context, b spec.Spec, e ast.ExprProviderIdentifier) (spec.ValueProviderIdentifier, []string, error) {
	name, nameChildren, err := in.Expr(ctx, b, e.Name)
	if err != nil {
		return spec.ValueProviderIdentifier{}, nil, err
	}

	nameStr, ok := name.(spec.ValueString)
	if !ok {
		return spec.ValueProviderIdentifier{}, nil, fmt.Errorf("provider name must be a string")
	}

	version, versionChildren, err := in.Expr(ctx, b, e.Version)
	if err != nil {
		return spec.ValueProviderIdentifier{}, nil, err
	}

	versionStr, ok := version.(spec.ValueString)
	if !ok {
		return spec.ValueProviderIdentifier{}, nil, fmt.Errorf("provider version must be a string")
	}

	children := append(nameChildren, versionChildren...)

	if e.Alias == "" {
		return spec.ValueProviderIdentifier{}, nil, fmt.Errorf("must provide alias")
	}

	return spec.ValueProviderIdentifier{
		Alias:   e.Alias,
		Name:    nameStr.Literal,
		Version: versionStr.Literal,
	}, children, nil
}

func (in Interpreter) resourceIdentifierExpr(ctx context.Context, b spec.Spec, e ast.ExprResourceIdentifier) (spec.ValueResourceIdentifier, []string, error) {
	val, children, err := in.Expr(ctx, b, e.Value)
	if err != nil {
		return spec.ValueResourceIdentifier{}, nil, err
	}

	if e.Alias == "" {
		return spec.ValueResourceIdentifier{}, nil, fmt.Errorf("must provide alias")
	}

	if e.ResourceType == "" {
		return spec.ValueResourceIdentifier{}, nil, fmt.Errorf("must provide resource type")
	}

	return spec.ValueResourceIdentifier{
		Alias:        e.Alias,
		ResourceType: e.ResourceType,
		Literal:      val,
	}, append(children, e.Alias), nil
}

func (in Interpreter) ioGetExpr(ctx context.Context, b spec.Spec, e ast.ExprIOGet) (spec.ValueUnresolved, []string, error) {
	objVal, children, err := in.Expr(ctx, b, e.Object)
	if err != nil {
		return spec.ValueUnresolved{}, nil, err
	}

	unresolved, ok := objVal.(spec.ValueUnresolved)
	if !ok {
		return spec.ValueUnresolved{}, nil, fmt.Errorf("property %q does not belong to unresolved object; use get", e.Name)
	}

	return spec.ValueUnresolved{
		Name:   e.Name,
		Object: unresolved,
	}, children, nil
}

func (in Interpreter) getExpr(ctx context.Context, b spec.Spec, e ast.ExprGet) (spec.Value, []string, error) {
	var m map[string]spec.Value

	objVal, children, err := in.Expr(ctx, b, e.Object)
	if err != nil {
		return nil, nil, err
	}

	switch obj := objVal.(type) {
	case spec.ValueMap:
		m = obj.Entries
	case spec.ValueResource:
		m = map[string]spec.Value{
			"identifier": obj.Identifier,
			"config":     obj.Config,
			"attrs":      obj.Attrs,
		}
	case spec.ValueUnresolved:
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

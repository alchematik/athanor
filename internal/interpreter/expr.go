package interpreter

import (
	"context"
	"fmt"
	"runtime"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/dependency"
	"github.com/alchematik/athanor/internal/repo"
	"github.com/alchematik/athanor/internal/spec"
)

func (in *Interpreter) Expr(ctx context.Context, ex ast.Expr) (spec.Value, []string, error) {
	switch e := ex.(type) {
	case ast.ExprString:
		return spec.ValueString{Literal: e.Value}, nil, nil
	case ast.ExprBool:
		return spec.ValueBool{Literal: e.Value}, nil, nil
	case ast.ExprMap:
		return in.mapExpr(ctx, e)
	case ast.ExprProvider:
		return in.provider(ctx, e)
	case ast.ExprResource:
		return in.resource(ctx, e)
	case ast.ExprResourceIdentifier:
		return in.resourceIdentifierExpr(ctx, e)
	case ast.ExprFile:
		return spec.ValueFile{
			Path: e.Path,
		}, nil, nil
	case ast.ExprGet:
		return in.getExpr(ctx, e)
	case ast.ExprGetRuntimeConfig:
		return spec.ValueRuntimeConfig{}, nil, nil
	case ast.ExprList:
		return in.listExpr(ctx, e)
	case ast.ExprNil:
		return spec.ValueNil{}, nil, nil
	default:
		return nil, nil, fmt.Errorf("unknown expr %T", ex)
	}
}

func (in *Interpreter) provider(ctx context.Context, e ast.ExprProvider) (spec.ValueProvider, []string, error) {
	var source any
	switch src := e.Source.(type) {
	case repo.PluginSourceLocal:
		source = dependency.SourceLocal{Path: src.Path}
	case repo.PluginSourceGitHubRelease:
		source = dependency.SourceGitHubRelease{
			RepoOwner: src.RepoOwner,
			RepoName:  src.RepoName,
			Name:      src.Name,
		}
	default:
		return spec.ValueProvider{}, nil, fmt.Errorf("unuspported source for provider: %T", e.Source)
	}

	dep := dependency.BinDependency{
		Type:   "provider",
		Source: source,
		OS:     runtime.GOOS,
		Arch:   runtime.GOARCH,
	}
	if _, err := in.DepManager.FetchBinDependency(ctx, dep); err != nil {
		return spec.ValueProvider{}, nil, err
	}

	return spec.ValueProvider{
		Repo: e.Source,
	}, nil, nil
}

func (in *Interpreter) resource(ctx context.Context, e ast.ExprResource) (spec.ValueResource, []string, error) {
	idVal, children, err := in.Expr(ctx, e.Identifier)
	if err != nil {
		return spec.ValueResource{}, nil, err
	}

	id, ok := idVal.(spec.ValueResourceIdentifier)
	if !ok {
		return spec.ValueResource{}, nil, fmt.Errorf("expected ResourceIdentifier, got %T", idVal)
	}

	configVal, configChildren, err := in.Expr(ctx, e.Config)
	if err != nil {
		return spec.ValueResource{}, nil, err
	}

	children = append(children, configChildren...)
	var out []string
	for _, child := range children {
		// Filter out alias to self.
		if child == id.Alias {
			continue
		}

		out = append(out, child)
	}

	return spec.ValueResource{
		Identifier: id,
		Config:     configVal,
		Attrs: spec.ValueUnresolved{
			Name: "attrs",
			Object: spec.ValueResourceRef{
				Alias: id.Alias,
			},
		},
	}, out, nil
}

func (in *Interpreter) mapExpr(ctx context.Context, e ast.ExprMap) (spec.ValueMap, []string, error) {
	m := spec.ValueMap{Entries: map[string]spec.Value{}}
	var children []string
	for k, v := range e.Entries {
		var err error
		var valChildren []string

		m.Entries[k], valChildren, err = in.Expr(ctx, v)
		if err != nil {
			return spec.ValueMap{}, nil, err
		}

		children = append(children, valChildren...)
	}

	return m, children, nil
}

func (in *Interpreter) listExpr(ctx context.Context, e ast.ExprList) (spec.ValueList, []string, error) {
	l := spec.ValueList{Elements: make([]spec.Value, len(e.Elements))}
	var children []string
	for i, v := range e.Elements {
		val, valChildren, err := in.Expr(ctx, v)
		if err != nil {
			return spec.ValueList{}, nil, err
		}

		children = append(children, valChildren...)
		l.Elements[i] = val
	}

	return l, children, nil
}

func (in *Interpreter) resourceIdentifierExpr(ctx context.Context, e ast.ExprResourceIdentifier) (spec.ValueResourceIdentifier, []string, error) {
	val, children, err := in.Expr(ctx, e.Value)
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

func (in *Interpreter) getExpr(ctx context.Context, e ast.ExprGet) (spec.ValueUnresolved, []string, error) {
	objVal, children, err := in.Expr(ctx, e.Object)
	if err != nil {
		return spec.ValueUnresolved{}, nil, err
	}

	// The object being nil means we're accessing a property on the environment,
	// like a resource.
	if _, isNil := objVal.(spec.ValueNil); isNil {
		children = append(children, e.Name)
	}

	return spec.ValueUnresolved{
		Name:   e.Name,
		Object: objVal,
	}, children, nil
}

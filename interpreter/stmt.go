package interpreter

import (
	"context"
	"fmt"
	"slices"

	"github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/spec"
)

func (in Interpreter) Stmt(ctx context.Context, b spec.Spec, st ast.Stmt) error {
	switch s := st.(type) {
	case ast.StmtProvider:
		return in.providerStmt(ctx, b, s)
	case ast.StmtResource:
		return in.resourceStmt(ctx, b, s)
	// case ast.StmtBlueprint:
	case ast.StmtBuild:
		return in.buildStmt(ctx, b, s)
	default:
		return fmt.Errorf("unknown stmt %T", st)
	}
}

// func (in Interpreter) blueprintStmt(ctx context.Context, b spec.Spec, s ast.StmtBlueprint) error {
// 	val, children, err := in.Expr(ctx, b, s.Expr)
// 	if err != nil {
// 		return err
// 	}
//
//
// }

func (in Interpreter) buildStmt(ctx context.Context, s spec.Spec, stmt ast.StmtBuild) error {
	subSpec := spec.Spec{
		Inputs:        map[string]spec.Value{},
		Providers:     map[string]spec.ValueProvider{},
		Resources:     map[string]spec.ValueResource{},
		DependencyMap: map[string][]string{},
		Components:    map[string]spec.Component{},
	}

	var children []string
	for name, expr := range stmt.Inputs {
		v, c, err := in.Expr(ctx, s, expr)
		if err != nil {
			return err
		}

		children = append(children, c...)
		subSpec.Inputs[name] = v
	}

	// TODO: May need to handle children here if allowing inside scope to access outside scope.
	_, _, err := in.Expr(ctx, subSpec, stmt.Blueprint)
	if err != nil {
		return err
	}

	s.DependencyMap[stmt.Alias] = append(s.DependencyMap[stmt.Alias], children...)
	s.Components[stmt.Alias] = spec.ComponentBuild{Spec: subSpec}

	return nil
}

func (in Interpreter) providerStmt(ctx context.Context, b spec.Spec, s ast.StmtProvider) error {
	val, _, err := in.Expr(ctx, b, s.Expr)
	if err != nil {
		return err
	}

	provider, ok := val.(spec.ValueProvider)
	if !ok {
		return fmt.Errorf("expected Provider type, got %T", val)
	}

	b.Providers[provider.Identifier.Alias] = provider

	return nil
}

func (in Interpreter) resourceStmt(ctx context.Context, b spec.Spec, s ast.StmtResource) error {
	val, children, err := in.Expr(ctx, b, s.Expr)
	if err != nil {
		return err
	}

	resource, ok := val.(spec.ValueResource)
	if !ok {
		return fmt.Errorf("expected Resource type, got %T", val)
	}

	alias := resource.Identifier.Alias
	b.DependencyMap[alias] = slices.Compact(append(b.DependencyMap[alias], children...))
	b.Resources[alias] = resource
	b.Providers[resource.Provider.Identifier.Alias] = resource.Provider
	b.Components[alias] = spec.ComponentResource{Value: resource}

	return nil
}

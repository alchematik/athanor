package interpreter

import (
	"context"

	"github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/build"
	"github.com/alchematik/athanor/build/component"
	"github.com/alchematik/athanor/build/value"
)

type Interpreter struct{}

func (in Interpreter) Interpret(ctx context.Context, bp ast.Blueprint) (build.Build, error) {
	b := build.Build{
		Providers:     map[string]value.Provider{},
		Resources:     map[string]value.Resource{},
		DependencyMap: map[string][]string{},
		Components:    map[string]component.Type{},
	}
	for _, st := range bp.Stmts {
		if err := in.Stmt(ctx, b, st); err != nil {
			return build.Build{}, err
		}
	}

	return b, nil
}

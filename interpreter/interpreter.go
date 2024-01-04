package interpreter

import (
	"context"

	"github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/spec"
)

type Interpreter struct{}

func (in Interpreter) Interpret(ctx context.Context, bp ast.Blueprint) (spec.Spec, error) {
	b := spec.Spec{
		Providers:     map[string]spec.ValueProvider{},
		Resources:     map[string]spec.ValueResource{},
		DependencyMap: map[string][]string{},
		Components:    map[string]spec.Component{},
	}
	for _, st := range bp.Stmts {
		if err := in.Stmt(ctx, b, st); err != nil {
			return spec.Spec{}, err
		}
	}

	return b, nil
}

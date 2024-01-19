package interpreter

import (
	"context"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/spec"
)

type Interpreter struct{}

func (in Interpreter) Interpret(ctx context.Context, blueprint ast.Blueprint) (spec.Spec, error) {
	s := spec.Spec{
		DependencyMap: map[string][]string{},
		Components:    map[string]spec.Component{},
	}

	for _, st := range blueprint.Stmts {
		if err := in.Stmt(ctx, s, st); err != nil {
			return spec.Spec{}, err
		}
	}

	return s, nil
}

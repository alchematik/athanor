package interpreter

import (
	"context"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/spec"
)

type Interpreter struct{}

func (in Interpreter) Interpret(ctx context.Context, s spec.Spec, build ast.StmtBuild) error {
	return in.Stmt(ctx, s, build)
}

package interpreter

import (
	"context"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/spec"
)

type Interpreter struct {
	Translator *plugin.Translator
}

func (in Interpreter) Interpret(ctx context.Context, s spec.Spec, build ast.StmtBuild) error {
	defer in.Translator.Stop()

	if err := in.Stmt(ctx, s, build); err != nil {
		return err
	}

	return nil
}

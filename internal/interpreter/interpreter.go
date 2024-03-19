package interpreter

import (
	"context"
	"fmt"
	"sync"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/dependency"
	"github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/spec"
)

type Interpreter struct {
	sync.Mutex

	PlugManager *plugin.Manager
	DepManager  *dependency.Manager

	stmts []Stmt
}

type Stmt struct {
	Spec spec.Spec
	Stmt ast.Stmt
}

func NewInterpreter(plugManager *plugin.Manager, depManager *dependency.Manager, s spec.Spec, build ast.StmtBuild) *Interpreter {
	return &Interpreter{
		PlugManager: plugManager,
		DepManager:  depManager,
		stmts: []Stmt{
			{
				Spec: s,
				Stmt: build,
			},
		},
	}
}

func (in *Interpreter) Next() []Stmt {
	in.Lock()
	defer in.Unlock()

	next := in.stmts
	in.stmts = []Stmt{}
	return next

}

func (in *Interpreter) Interpret(ctx context.Context, stmt Stmt) error {
	switch s := stmt.Stmt.(type) {
	case ast.StmtResource:
		return in.resourceStmt(ctx, stmt.Spec, s)
	case ast.StmtBuild:
		return in.buildStmt(ctx, stmt.Spec, s)
	default:
		return fmt.Errorf("unknown stmt %T", stmt.Stmt)
	}
}

package ast

import (
	"github.com/alchematik/athanor/internal/repo"
)

type Stmt interface {
	isStmtType()
}

type StmtResource struct {
	Stmt

	Exists   Expr
	Expr     ExprResource
	Provider ExprProvider
}

type StmtBuild struct {
	Stmt

	Translator Translator
	Build      ExprBuild
}

type Translator struct {
	Source repo.PluginSource
}

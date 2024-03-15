package ast

import (
	"github.com/alchematik/athanor/internal/repo"
)

type Stmt interface {
	isStmtType()
}

type StmtResource struct {
	Stmt

	Expr Expr
}

type StmtBuild struct {
	Stmt

	Translator Translator
	Build      ExprBuild
}

type Translator struct {
	Source repo.Source
}

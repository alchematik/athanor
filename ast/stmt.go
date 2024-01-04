package ast

type Stmt interface {
	isStmtType()
}

type StmtProvider struct {
	Stmt

	Expr Expr
}

type StmtResource struct {
	Stmt

	Expr Expr
}

type StmtBlueprint struct {
	Stmt

	Alias string
	Expr  Expr
}

type StmtBuild struct {
	Stmt

	Alias string
	Expr  Expr
}

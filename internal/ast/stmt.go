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

// type StmtBlueprint struct {
// 	Stmt
//
// 	Alias string
// 	Expr  Expr
// }

type StmtBuild struct {
	Stmt

	Repo       Repo
	Translator Translator

	Alias     string
	Inputs    map[string]Expr
	Blueprint Expr
}

type Translator struct {
	Name    string
	Version string
	Repo    Repo
}

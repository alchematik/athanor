package ast

type Expr interface {
	isExprExpr() bool
}

type ExprBlueprint struct {
	Expr

	Stmts []Stmt
}

type ExprString struct {
	Expr

	Value string
}

type ExprBool struct {
	Expr

	Value bool
}

type ExprFile struct {
	Expr

	Path string
}

type ExprMap struct {
	Expr

	Entries map[string]Expr
}

type ExprList struct {
	Expr

	Elements []Expr
}

type ExprGet struct {
	Expr

	Name   string
	Object Expr
}

type ExprGetRuntimeConfig struct {
	Expr
}

type ExprProvider struct {
	Expr

	Name    string
	Version string
	Repo    Repo
}

type Repo interface {
	isRepo()
}

type RepoLocal struct {
	Repo

	Path string
}

type ExprResource struct {
	Expr

	Provider   Expr
	Identifier Expr
	Config     Expr
	Exists     Expr
}

type ExprResourceIdentifier struct {
	Expr

	Alias        string
	ResourceType string
	Value        Expr
}

type ExprNil struct {
	Expr
}

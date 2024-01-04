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

type ExprMap struct {
	Expr

	Entries map[string]Expr
}

type ExprGet struct {
	Expr

	Name   string
	Object Expr
}

type ExprIOGet struct {
	Expr

	Name   string
	Object Expr
}

type ExprGetProvider struct {
	Expr

	Alias string
}

type ExprGetResource struct {
	Expr

	Alias string
}

type ExprProvider struct {
	Expr

	Identifier Expr
}

type ExprProviderIdentifier struct {
	Expr

	Alias   string
	Name    Expr
	Version Expr
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

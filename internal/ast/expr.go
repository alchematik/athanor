package ast

import (
	"github.com/alchematik/athanor/internal/repo"
)

type Expr interface {
	isExprExpr() bool
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

	Source repo.PluginSource
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

type ExprBuild struct {
	Expr

	Alias         string
	Source        repo.BlueprintSource
	Config        []Expr
	RuntimeConfig Expr
}

type ExprNil struct {
	Expr
}

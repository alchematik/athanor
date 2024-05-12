package ast

import (
	"fmt"

	external "github.com/alchematik/athanor/ast"
)

func ConvertExpr(ctx Context, name string, expr external.Expr) (any, error) {
	switch expr.Value.(type) {
	case external.StringLiteral:
		return ConvertStringExpr(ctx, name, expr)
	case external.BoolLiteral:
		return ConvertBoolExpr(ctx, name, expr)
	case external.MapCollection:
		return ConvertMapExpr(ctx, name, expr)
	case external.Resource:
		return ConvertResourceExpr(ctx, name, expr)
	// case external.LocalFile:
	// return ConvertFileExpr(expr)
	default:
		return nil, fmt.Errorf("invalid expr: %T", expr.Value)
	}
}

func ConvertStringExpr(ctx Context, name string, expr external.Expr) (ExprString, error) {
	switch value := expr.Value.(type) {
	case external.StringLiteral:
		return StringLiteral(value.Value), nil
	default:
		return nil, fmt.Errorf("invalid bool expr: %T", expr)
	}
}

func ConvertBoolExpr(ctx Context, name string, expr external.Expr) (ExprBool, error) {
	switch value := expr.Value.(type) {
	case external.BoolLiteral:
		return BoolLiteral(value.Value), nil
	default:
		return nil, fmt.Errorf("invalid bool expr: %T", expr)
	}
}

func ConvertMapExpr(ctx Context, name string, expr external.Expr) (ExprMap, error) {
	switch value := expr.Value.(type) {
	case external.MapCollection:
		m := MapCollection{}
		for k, v := range value.Value {
			e, err := ConvertExpr(ctx, name, v)
			if err != nil {
				return nil, err
			}
			m[k] = e
		}
		return m, nil
	default:
		return nil, fmt.Errorf("%s: invalid map expr: %T", name, expr)
	}
}

func ConvertResourceExpr(ctx Context, name string, expr external.Expr) (ExprResource, error) {
	switch value := expr.Value.(type) {
	case external.Resource:
		identifier, err := ConvertExpr(ctx, name, value.Identifier)
		if err != nil {
			return nil, err
		}

		config, err := ConvertExpr(ctx, name, value.Config)
		if err != nil {
			return nil, err
		}

		exists, err := ConvertBoolExpr(ctx, name, value.Exists)
		if err != nil {
			return nil, err
		}

		return Resource{
			Identifier: identifier,
			Config:     config,
			Exists:     exists,
		}, nil
	case external.GetResource:
		from, err := ConvertExpr(ctx, name, value.From)
		if err != nil {
			return nil, err
		}

		return GetResource{Name: value.Name, From: from}, nil
	default:
		return nil, fmt.Errorf("invalid resource expr: %T", expr)
	}
}

// func ConvertLocalFileExpr(expr any)

type StringLiteral string

func (s StringLiteral) Eval(_ Context) (string, error) {
	return string(s), nil
}

type BoolLiteral bool

func (b BoolLiteral) Eval(_ Context) (bool, error) {
	return bool(b), nil
}

type MapCollection map[string]any

func (m MapCollection) Eval(_ Context) (map[string]any, error) {
	return m, nil
}

type Resource struct {
	Provider   ExprProvider
	Exists     ExprBool
	Identifier any
	Config     any
}

func (r Resource) Eval(_ Context) (Resource, error) {
	return r, nil
}

type Provider struct {
}

type GetResource struct {
	Name string
	From any
}

func (g GetResource) Eval(ctx Context) (Resource, error) {
	// TODO: Handle "from".
	// return ctx.GetResource(g.Name)
	return Resource{}, nil
}

type Build struct {
	RuntimeInput ExprMap
	Blueprint    Blueprint
}

func (b Build) Eval(ctx Context) (Build, error) {
	return b, nil
}

type LocalFile struct {
	Path string
}

type Blueprint struct {
	Stmts []Stmt
}

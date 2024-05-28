package ast

import (
	"fmt"

	external "github.com/alchematik/athanor/ast"
)

func ConvertAnyExpr(scope *Scope, name string, expr external.Expr) (Expr[any], error) {
	switch expr.Value.(type) {
	case external.StringLiteral:
		return ConvertStringExpr[any](scope, name, expr)
	case external.BoolLiteral:
		return ConvertBoolExpr[any](scope, name, expr)
	// case external.MapCollection:
	// 	return ConvertMapExpr(scope, name, expr)
	// case external.Resource:
	// 	return ConvertResourceExpr(scope, name, expr)
	// case external.LocalFile:
	// return ConvertFileExpr(expr)
	case external.MapCollection:
		return ConvertMapExpr[any](scope, name, expr)
	case external.Resource:
		return ResourceExpr[any]{
			// Exists:

		}, nil
	default:
		return nil, fmt.Errorf("invalid expr: %T", expr.Value)
	}
}

func ConvertStringExpr[T any | string](scope *Scope, name string, expr external.Expr) (Expr[T], error) {
	switch value := expr.Value.(type) {
	case external.StringLiteral:
		var val T
		switch v := any(&val).(type) {
		case *string:
			*v = value.Value
		case *any:
			*v = value.Value
		default:
			return nil, fmt.Errorf("unsupported string type: %T", val)
		}
		return Literal[T]{Value: val}, nil
	default:
		return nil, fmt.Errorf("invalid bool expr: %T", expr)
	}
}

type Literal[T any] struct {
	Value T
}

func (l Literal[T]) Eval(_ *Scope) (T, error) {
	return l.Value, nil
}

type Map[T any] struct {
	Value map[Expr[string]]Expr[any]
}

func (m Map[T]) Eval(scope *Scope) (T, error) {
	out := map[string]any{}
	var val T
	for k, v := range m.Value {
		outKey, err := k.Eval(scope)
		if err != nil {
			return val, err
		}

		outVal, err := v.Eval(scope)
		if err != nil {
			return val, err
		}

		out[outKey] = outVal
	}

	switch v := any(&val).(type) {
	case *any:
		*v = out
	case *map[string]any:
		*v = out
	}

	return val, nil
}

type ResourceExpr[T any | Resource] struct {
	Exists     Expr[bool]
	Identifier Expr[any]
	Config     Expr[any]
}

func (r ResourceExpr[T]) Eval(scope *Scope) (T, error) {
	var out T

	e, err := r.Exists.Eval(scope)
	if err != nil {
		return out, err
	}

	id, err := r.Identifier.Eval(scope)
	if err != nil {
		return out, err
	}

	c, err := r.Config.Eval(scope)
	if err != nil {
		return out, err
	}

	resource := Resource{
		Exists:     e,
		Identifier: id,
		Config:     c,
	}

	switch o := any(&out).(type) {
	case *Resource:
		*o = resource
	case *any:
		*o = resource
	}

	return out, nil
}

func ConvertBoolExpr[T any | bool](scope *Scope, name string, expr external.Expr) (Expr[T], error) {
	switch value := expr.Value.(type) {
	case external.BoolLiteral:
		var val T
		switch v := any(&value).(type) {
		case *bool:
			*v = value.Value
		case *any:
			*v = value.Value
		}
		return Literal[T]{Value: val}, nil
	default:
		return nil, fmt.Errorf("invalid bool expr: %T", expr)
	}
}

func ConvertMapExpr[T any | map[string]any](scope *Scope, name string, expr external.Expr) (Expr[T], error) {
	switch value := expr.Value.(type) {
	case external.MapCollection:
		m := Map[T]{
			Value: map[Expr[string]]Expr[any]{},
		}
		for k, v := range value.Value {
			key, err := ConvertStringExpr[string](scope, name, external.Expr{Value: external.StringLiteral{Value: k}})
			if err != nil {
				return nil, err
			}

			val, err := ConvertAnyExpr(scope, name, v)
			if err != nil {
				return nil, err
			}

			m.Value[key] = val
		}
		return m, nil
	default:
		return nil, fmt.Errorf("%s: invalid map expr: %T", name, expr)
	}
}

func ConvertResourceExpr(scope *Scope, name string, expr external.Expr) (Expr[Resource], error) {
	switch value := expr.Value.(type) {
	case external.Resource:
		identifier, err := ConvertAnyExpr(scope, name, value.Identifier)
		if err != nil {
			return nil, err
		}

		config, err := ConvertAnyExpr(scope, name, value.Config)
		if err != nil {
			return nil, err
		}

		exists, err := ConvertBoolExpr[bool](scope, name, value.Exists)
		if err != nil {
			return nil, err
		}

		return ResourceExpr[Resource]{
			Identifier: identifier,
			Config:     config,
			Exists:     exists,
		}, nil
	case external.GetResource:
		from, err := ConvertAnyExpr(scope, name, value.From)
		if err != nil {
			return nil, err
		}

		return GetResource{Name: value.Name, From: from}, nil
	default:
		return nil, fmt.Errorf("invalid resource expr: %T", expr)
	}
}

type MapCollection map[string]ExprAny

func (m MapCollection) Eval(scope *Scope) (map[string]any, error) {
	out := map[string]any{}
	for k, v := range m {
		o, err := v.Eval(scope)
		if err != nil {
			return nil, err
		}

		out[k] = o
	}

	return out, nil
}

type Resource struct {
	Provider   Provider
	Exists     bool
	Identifier any
	Config     any
}

type Provider struct {
	Name    string
	Version string
}

type GetResource struct {
	Name string
	From any
}

func (g GetResource) Eval(scope *Scope) (Resource, error) {
	// TODO: Handle "from".
	// return scope.GetResource(g.Name)
	return Resource{}, nil
}

type Build struct {
	RuntimeInput ExprMap
	Blueprint    Blueprint
}

func (b Build) Eval(scope *Scope) (Build, error) {
	return b, nil
}

type LocalFile struct {
	Path string
}

type Blueprint struct {
	Stmts []Stmt
}

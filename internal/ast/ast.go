package ast

import (
	"fmt"
	"log/slog"

	external "github.com/alchematik/athanor/ast"
)

type Converter struct {
	Logger               *slog.Logger
	BlueprintInterpreter BlueprintInterpreter
}

type BlueprintInterpreter interface {
	InterpretBlueprint(source external.BlueprintSource, input map[string]any) (external.Blueprint, error)
}

func (c *Converter) ConvertStmt(scope *Scope, stmt external.Stmt) (Stmt, error) {
	switch stmt := stmt.Value.(type) {
	case external.DeclareBuild:
		return c.ConvertBuildStmt(scope, stmt)
	case external.DeclareResource:
		return c.ConvertResourceStmt(scope, stmt)
	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

func (c *Converter) ConvertResourceStmt(scope *Scope, stmt external.DeclareResource) (StmtResource, error) {
	resource, err := ConvertResourceExpr(scope, stmt.Name, stmt.Resource)
	if err != nil {
		return StmtResource{}, err
	}

	r := StmtResource{
		Name:     stmt.Name,
		Resource: resource,
	}
	scope.SetResource(r)
	return r, nil
}

func (c *Converter) ConvertBuildStmt(scope *Scope, build external.DeclareBuild) (StmtBuild, error) {
	blueprint, err := c.BlueprintInterpreter.InterpretBlueprint(build.BlueprintSource, build.Input)
	if err != nil {
		return StmtBuild{}, err
	}

	runtimeInput, err := ConvertMapExpr[map[string]any](scope, build.Name, build.Runtimeinput)
	if err != nil {
		return StmtBuild{}, fmt.Errorf("converting runtime input: %s", err)
	}

	c.Logger.Info("converting blueprint >>", "blueprint", blueprint)

	var stmts []Stmt
	subScope := scope.NewSubScope(build.Name)
	for _, stmt := range blueprint.Stmts {
		c.Logger.Info("converting statement >>>>>>>>>", "stmt", stmt)
		s, err := c.ConvertStmt(subScope, stmt)
		if err != nil {
			c.Logger.Info(">>>>>>>>>>>>>>>>>>>>>", "err", err)
			return StmtBuild{}, err
		}

		stmts = append(stmts, s)
	}

	b := StmtBuild{
		Name: build.Name,
		Build: Build{
			RuntimeInput: runtimeInput,
			Blueprint: Blueprint{
				Stmts: stmts,
			},
		},
	}

	scope.SetBuild(b)

	return b, nil
}

type Stmt interface {
	Eval(*Scope) error
}

type StmtBuild struct {
	Name  string
	Build Build
}

func (s StmtBuild) Eval(*Scope) error {
	return nil
}

type StmtResource struct {
	Name     string
	Resource Expr[Resource]
}

func (s StmtResource) Eval(*Scope) error {
	return nil
}

type StmtWatcher struct {
	Name  string
	Value any
}

type Expr[T any] interface {
	Eval(*Scope) (T, error)
}

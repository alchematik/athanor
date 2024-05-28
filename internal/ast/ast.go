package ast

import (
	"fmt"
	"log/slog"
	"strings"

	external "github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/internal/dag"
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

func NewScope(name string) *Scope {
	return &Scope{
		name:      name,
		resources: map[string]string{},
		builds:    map[string]string{},
		dag:       dag.NewGraph(),
		sub:       map[string]*Scope{},
	}
}

type Scope struct {
	name      string
	resources map[string]string
	builds    map[string]string
	sub       map[string]*Scope
	dag       *dag.Graph
}

func (c *Scope) SetResource(stmt StmtResource) {
	id := strings.Join([]string{c.name, stmt.Name}, ".")
	c.resources[stmt.Name] = id
	c.dag.AddEdge(c.name, id)
}

func (c *Scope) SetBuild(stmt StmtBuild) {
	id := strings.Join([]string{c.name, stmt.Name}, ".")
	c.builds[stmt.Name] = id
	c.dag.AddEdge(c.name, id)
}

func (c *Scope) SetVars(vars map[string]any) {

}

func (c *Scope) NewSubScope(name string) *Scope {
	id := strings.Join([]string{c.name, name}, ".")
	sub := NewScope(id)
	sub.dag = c.dag
	sub.name = id
	c.sub[name] = sub

	return sub
}

func (c *Scope) Resources() []string {
	keys := make([]string, 0, len(c.resources))
	for k := range c.resources {
		keys = append(keys, k)
	}

	return keys
}

func (c *Scope) Builds() []string {
	keys := make([]string, 0, len(c.builds))
	for k := range c.builds {
		keys = append(keys, k)
	}

	return keys
}

func (c *Scope) SubContext(name string) *Scope {
	return c.sub[name]
}

func (c *Scope) DAG() *dag.Graph {
	return c.dag
}

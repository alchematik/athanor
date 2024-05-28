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

func (c *Converter) ConvertStmt(scope Scope, stmt external.Stmt) (Stmt, error) {
	switch stmt := stmt.Value.(type) {
	case external.DeclareBuild:
		return c.ConvertBuildStmt(scope, stmt)
	case external.DeclareResource:
		return c.ConvertResourceStmt(scope, stmt)
	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

func (c *Converter) ConvertResourceStmt(scope Scope, stmt external.DeclareResource) (StmtResource, error) {
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

func (c *Converter) ConvertBuildStmt(scope Scope, build external.DeclareBuild) (StmtBuild, error) {
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
	subScope := scope.NewSubContext(build.Name)
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
	Eval(Scope) error
}

type StmtBuild struct {
	Name  string
	Build Build
}

func (s StmtBuild) Eval(Scope) error {
	return nil
}

type StmtResource struct {
	Name     string
	Resource ExprResource
}

func (s StmtResource) Eval(Scope) error {
	return nil
}

type StmtWatcher struct {
	Name  string
	Value any
}

type Expr[T any] interface {
	Eval(Scope) (T, error)
}

type ExprAny interface {
	Eval(Scope) (any, error)
}

type ExprBool interface {
	Eval(Scope) (bool, error)
}

type ExprString interface {
	Eval(Scope) (string, error)
}

type ExprResource interface {
	Eval(Scope) (Resource, error)
}

type ExprProvider interface {
	Eval(Scope) (Provider, error)
}

type ExprBuild interface {
	Eval(Scope) (Build, error)
}

type ExprMap interface {
	Eval(Scope) (map[string]any, error)
}

type ExprBlueprint interface {
	Eval(Scope) (Blueprint, error)
}

type ExprBlueprintSource interface {
	// Convert()
}

// TODO: rename this to "Scope"
type Scope interface {
	SetResource(StmtResource)
	SetBuild(StmtBuild)
	SetVars(map[string]any)
	NewSubContext(string) Scope
	DAG() *dag.Graph
	Resources() []string
	Builds() []string
	SubContext(string) Scope
}

func NewScope(name string) *ScopeImpl {
	return &ScopeImpl{
		name:      name,
		resources: map[string]string{},
		builds:    map[string]string{},
		dag:       dag.NewGraph(),
		sub:       map[string]Scope{},
	}
}

type ScopeImpl struct {
	name      string
	resources map[string]string
	builds    map[string]string
	sub       map[string]Scope
	dag       *dag.Graph
}

func (c *ScopeImpl) SetResource(stmt StmtResource) {
	id := strings.Join([]string{c.name, stmt.Name}, ".")
	c.resources[stmt.Name] = id
	c.dag.AddEdge(c.name, id)
}

func (c *ScopeImpl) SetBuild(stmt StmtBuild) {
	id := strings.Join([]string{c.name, stmt.Name}, ".")
	c.builds[stmt.Name] = id
	c.dag.AddEdge(c.name, id)
}

func (c *ScopeImpl) SetVars(vars map[string]any) {

}

func (c *ScopeImpl) NewSubContext(name string) Scope {
	id := strings.Join([]string{c.name, name}, ".")
	sub := NewScope(id)
	sub.dag = c.dag
	sub.name = id
	c.sub[name] = sub

	return sub
}

func (c *ScopeImpl) Resources() []string {
	keys := make([]string, 0, len(c.resources))
	for k := range c.resources {
		keys = append(keys, k)
	}

	return keys
}

func (c *ScopeImpl) Builds() []string {
	keys := make([]string, 0, len(c.builds))
	for k := range c.builds {
		keys = append(keys, k)
	}

	return keys
}

func (c *ScopeImpl) SubContext(name string) Scope {
	return c.sub[name]
}

func (c *ScopeImpl) DAG() *dag.Graph {
	return c.dag
}

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

func (c *Converter) ConvertStmt(ctx Context, stmt external.Stmt) (Stmt, error) {
	switch stmt := stmt.Value.(type) {
	case external.DeclareBuild:
		return c.ConvertBuildStmt(ctx, stmt)
	case external.DeclareResource:
		return c.ConvertResourceStmt(ctx, stmt)
	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

func (c *Converter) ConvertResourceStmt(ctx Context, stmt external.DeclareResource) (StmtResource, error) {
	resource, err := ConvertResourceExpr(ctx, stmt.Name, stmt.Resource)
	if err != nil {
		return StmtResource{}, err
	}

	r := StmtResource{
		Name:     stmt.Name,
		Resource: resource,
	}
	ctx.SetResource(r)
	return r, nil
}

func (c *Converter) ConvertBuildStmt(ctx Context, build external.DeclareBuild) (StmtBuild, error) {
	blueprint, err := c.BlueprintInterpreter.InterpretBlueprint(build.BlueprintSource, build.Input)
	if err != nil {
		return StmtBuild{}, err
	}

	runtimeInput, err := ConvertMapExpr[map[string]any](ctx, build.Name, build.Runtimeinput)
	if err != nil {
		return StmtBuild{}, fmt.Errorf("converting runtime input: %s", err)
	}

	c.Logger.Info("converting blueprint >>", "blueprint", blueprint)

	var stmts []Stmt
	subCtx := ctx.NewSubContext(build.Name)
	for _, stmt := range blueprint.Stmts {
		c.Logger.Info("converting statement >>>>>>>>>", "stmt", stmt)
		s, err := c.ConvertStmt(subCtx, stmt)
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

	ctx.SetBuild(b)

	return b, nil
}

type Stmt interface {
	Eval(Context) error
}

type StmtBuild struct {
	Name  string
	Build Build
}

func (s StmtBuild) Eval(Context) error {
	return nil
}

type StmtResource struct {
	Name     string
	Resource ExprResource
}

func (s StmtResource) Eval(Context) error {
	return nil
}

type StmtWatcher struct {
	Name  string
	Value any
}

type Expr[T any] interface {
	Eval(Context) (T, error)
}

type ExprAny interface {
	Eval(Context) (any, error)
}

type ExprBool interface {
	Eval(Context) (bool, error)
}

type ExprString interface {
	Eval(Context) (string, error)
}

type ExprResource interface {
	Eval(Context) (Resource, error)
}

type ExprProvider interface {
	Eval(Context) (Provider, error)
}

type ExprBuild interface {
	Eval(Context) (Build, error)
}

type ExprMap interface {
	Eval(Context) (map[string]any, error)
}

type ExprBlueprint interface {
	Eval(Context) (Blueprint, error)
}

type ExprBlueprintSource interface {
	// Convert()
}

// TODO: rename this to "Scope"
type Context interface {
	SetResource(StmtResource)
	SetBuild(StmtBuild)
	SetVars(map[string]any)
	NewSubContext(string) Context
	DAG() *dag.Graph
	Resources() []string
	Builds() []string
	SubContext(string) Context
}

func NewContext(name string) *ContextImpl {
	return &ContextImpl{
		name:      name,
		resources: map[string]string{},
		builds:    map[string]string{},
		dag:       dag.NewGraph(),
		sub:       map[string]Context{},
	}
}

type ContextImpl struct {
	name      string
	resources map[string]string
	builds    map[string]string
	sub       map[string]Context
	dag       *dag.Graph
}

func (c *ContextImpl) SetResource(stmt StmtResource) {
	id := strings.Join([]string{c.name, stmt.Name}, ".")
	c.resources[stmt.Name] = id
	c.dag.AddNode(id, stmt)
	c.dag.AddEdge(c.name, id)
}

func (c *ContextImpl) SetBuild(stmt StmtBuild) {
	id := strings.Join([]string{c.name, stmt.Name}, ".")
	c.builds[stmt.Name] = id
	c.dag.AddNode(id, stmt)
	c.dag.AddEdge(c.name, id)
}

func (c *ContextImpl) SetVars(vars map[string]any) {

}

func (c *ContextImpl) NewSubContext(name string) Context {
	id := strings.Join([]string{c.name, name}, ".")
	sub := NewContext(id)
	sub.dag = c.dag
	sub.name = id
	c.sub[name] = sub

	return sub
}

func (c *ContextImpl) Resources() []string {
	keys := make([]string, 0, len(c.resources))
	for k := range c.resources {
		keys = append(keys, k)
	}

	return keys
}

func (c *ContextImpl) Builds() []string {
	keys := make([]string, 0, len(c.builds))
	for k := range c.builds {
		keys = append(keys, k)
	}

	return keys
}

func (c *ContextImpl) SubContext(name string) Context {
	return c.sub[name]
}

func (c *ContextImpl) DAG() *dag.Graph {
	return c.dag
}

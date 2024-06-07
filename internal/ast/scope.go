package ast

import (
	"fmt"

	"github.com/alchematik/athanor/internal/dag"
	"github.com/alchematik/athanor/internal/set"
)

func NewScope() *Scope {
	return &Scope{
		resources: set.NewSet[string](),
		sub:       map[string]*Scope{},
	}
}

func NewGlobal() *Global {
	return &Global{
		components: map[string]any{},
		dag:        dag.NewGraph(),
		Scope:      NewScope(),
	}
}

type Global struct {
	id         string
	components map[string]any
	dag        *dag.Graph
	Scope      *Scope
}

func (g *Global) ComponentID(name string) string {
	return fmt.Sprintf("%s.%s", g.id, name)
}

func (g *Global) SetEvaluable(id string, e any) string {
	g.components[id] = e
	g.dag.AddEdge(g.id, id)
	return id
}

func (g *Global) SetResource(id string, e any) {
	g.components[id] = e
	g.dag.AddEdge(g.id, id)
	g.Scope.resources.Add(id)
}

func (g *Global) SetBuild(id string, e any) {
	g.components[id] = e
	g.dag.AddEdge(g.id, id)
}

func (g *Global) Sub(name string) *Global {
	subID := fmt.Sprintf("%s.%s", g.id, name)
	subScope := NewScope()
	g.Scope.sub[subID] = subScope
	return &Global{
		id:         subID,
		components: g.components,
		dag:        g.dag,
		Scope:      subScope,
	}
}

type Scope struct {
	resources *set.Set[string]
	sub       map[string]*Scope
}

func (c *Scope) RegisterResource(id string) {
	c.resources.Add(id)
}

func (c *Scope) RegisterBuild(id string) *Scope {
	sub := NewScope()
	c.sub[id] = sub

	return sub
}

func (c *Scope) Resources() []string {
	return c.resources.Values()
}

func (c *Scope) Builds() []string {
	builds := make([]string, 0, len(c.sub))
	for k := range c.sub {
		builds = append(builds, k)
	}

	return builds
}

func (c *Scope) Sub(id string) *Scope {
	return c.sub[id]
}

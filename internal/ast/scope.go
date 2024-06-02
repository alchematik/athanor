package ast

import (
	"fmt"

	"github.com/alchematik/athanor/internal/dag"
	"github.com/alchematik/athanor/internal/state"
)

func NewScope(name string) *Scope {
	return &Scope{
		name:      name,
		resources: map[string]string{},
		sub:       map[string]*Scope{},
	}
}

type Evaluable interface {
	Eval(*state.State) error
}

func NewGlobal() *Global {
	return &Global{
		components: map[string]Evaluable{},
		dag:        dag.NewGraph(),
	}
}

type Global struct {
	id string

	components map[string]Evaluable

	dag *dag.Graph
}

func (g *Global) ComponentID(name string) string {
	return fmt.Sprintf("%s.%s", g.id, name)
}

func (g *Global) SetEvaluable(id string, e Evaluable) string {
	g.components[id] = e
	g.dag.AddEdge(g.id, id)
	return id
}

func (g *Global) Sub(name string) *Global {
	return &Global{
		id:         fmt.Sprintf("%s.%s", g.id, name),
		components: g.components,
		dag:        g.dag,
	}
}

type Scope struct {
	name string
	// id -> name?
	resources map[string]string
	sub       map[string]*Scope
}

func (c *Scope) Name() string {
	return c.name
}

func (c *Scope) SetResource(id, name string) {
	c.resources[id] = name
}

func (c *Scope) SetBuild(id, name string) *Scope {
	sub := NewScope(name)
	c.sub[name] = sub

	return sub
}

func (c *Scope) Resources() []string {
	names := make([]string, 0, len(c.resources))
	for _, v := range c.resources {
		names = append(names, v)
	}

	return names
}

func (c *Scope) Builds() []*Scope {
	builds := make([]*Scope, 0, len(c.sub))
	for _, b := range c.sub {
		builds = append(builds, b)
	}

	return builds
}

package ast

import (
	"strings"

	"github.com/alchematik/athanor/internal/dag"
)

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

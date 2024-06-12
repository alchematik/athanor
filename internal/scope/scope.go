package scope

import (
	"fmt"

	"github.com/alchematik/athanor/internal/dag"
	"github.com/alchematik/athanor/internal/set"
)

func NewBuild() *Build {
	return &Build{
		resources: set.NewSet[string](),
		builds:    map[string]*Build{},
	}
}

func NewScope() *Scope {
	return &Scope{
		components: map[string]any{},
		dag:        dag.NewGraph(),
		build:      NewBuild(),
	}
}

type Scope struct {
	id         string
	components map[string]any
	dag        *dag.Graph
	build      *Build
}

func (g *Scope) ComponentID(name string) string {
	return fmt.Sprintf("%s.%s", g.id, name)
}

func (g *Scope) SetBuild(id string, e any) string {
	g.components[id] = e
	if g.id == "" {
		return ""
	}
	g.dag.AddEdge(g.id, id)
	return id
}

func (g *Scope) SetResource(id string, e any) {
	g.components[id] = e
	if g.id == "" {
		return
	}
	g.dag.AddEdge(g.id, id)
	g.build.resources.Add(id)
}

func (g *Scope) Component(id string) (any, bool) {
	comp, ok := g.components[id]
	return comp, ok
}

func (g *Scope) Sub(name string) *Scope {
	subID := fmt.Sprintf("%s.%s", g.id, name)
	subBuild := NewBuild()
	g.build.builds[subID] = subBuild
	return &Scope{
		id:         subID,
		components: g.components,
		dag:        g.dag,
		build:      subBuild,
	}
}

func (g *Scope) Build() *Build {
	return g.build
}

func (g *Scope) NewIterator() *dag.Iterator {
	return dag.InitIterator(g.dag)
}

type Build struct {
	resources *set.Set[string]
	builds    map[string]*Build
}

func (c *Build) RegisterResource(id string) {
	c.resources.Add(id)
}

func (c *Build) RegisterBuild(id string) *Build {
	sub := NewBuild()
	c.builds[id] = sub

	return sub
}

func (c *Build) Resources() []string {
	return c.resources.Values()
}

func (c *Build) Builds() []string {
	builds := make([]string, 0, len(c.builds))
	for k := range c.builds {
		builds = append(builds, k)
	}

	return builds
}

func (c *Build) Build(id string) *Build {
	return c.builds[id]
}

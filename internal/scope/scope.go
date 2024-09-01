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

func NewRootScope() *Scope {
	return &Scope{
		components: map[string]any{},
		dag:        dag.NewGraph(),
		build:      NewBuild(),
	}
}

type Scope struct {
	id         string
	components map[string]any
	build      *Build

	dag *dag.Graph

	// resources is a map of build ID to child resources.
	resources map[string][]string

	// builds is a map of build ID to child builds.
	builds map[string][]string
}

func (s *Scope) ID() string {
	return s.id
}

func (s *Scope) ComponentID(name string) string {
	return fmt.Sprintf("%s.%s", s.id, name)
}

func (s *Scope) SetBuild(id string, e any) {
	s.components[id] = e

	// Root scope will not have an ID.
	if s.id == "" {
		return
	}

	s.dag.AddEdge(s.id, id)
}

func (s *Scope) SetResource(id string, e any) {
	s.components[id] = e

	// Root scope will not have an ID.
	if s.id == "" {
		return
	}

	s.dag.AddEdge(s.id, id)
	s.build.resources.Add(id)
}

func (s *Scope) Component(id string) (any, bool) {
	comp, ok := s.components[id]
	return comp, ok
}

func (s *Scope) Sub(name string) *Scope {
	subID := fmt.Sprintf("%s.%s", s.id, name)
	subBuild := NewBuild()
	s.build.builds[subID] = subBuild
	return &Scope{
		id:         subID,
		components: s.components,
		dag:        s.dag,
		build:      subBuild,
	}
}

func (s *Scope) Build() *Build {
	return s.build
}

func (s *Scope) NewIterator() *dag.Iterator {
	return dag.InitIterator(s.dag)
}

type Build struct {
	resources *set.Set[string]
	builds    map[string]*Build
}

func (b *Build) Resources() []string {
	return b.resources.Values()
}

func (b *Build) Builds() []string {
	builds := make([]string, 0, len(b.builds))
	for k := range b.builds {
		builds = append(builds, k)
	}

	return builds
}

func (b *Build) Build(id string) *Build {
	return b.builds[id]
}

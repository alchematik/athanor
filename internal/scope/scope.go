package scope

import (
	"github.com/alchematik/athanor/internal/dag"
	"github.com/alchematik/athanor/internal/set"
)

func NewScope() *Scope {
	return &Scope{
		components: map[string]any{},
		dag:        dag.NewGraph(),
		resources:  map[string]*set.Set[string]{},
		builds:     map[string]*set.Set[string]{},
	}
}

type Scope struct {
	components map[string]any
	dag        *dag.Graph

	// resources is a map of build ID to child resources.
	resources map[string]*set.Set[string]

	// builds is a map of build ID to child builds.
	builds map[string]*set.Set[string]
}

func (s *Scope) SetBuild(parent, id string, e any) {
	s.components[id] = e

	existing, ok := s.builds[parent]
	if !ok {
		existing = set.NewSet[string]()
		s.builds[parent] = existing
	}

	existing.Add(id)

	// Root scope will not have an ID.
	if parent == "" {
		return
	}

	s.dag.AddEdge(parent, id)
}

func (s *Scope) SetResource(parent, id string, e any) {
	s.components[id] = e

	existing, ok := s.resources[parent]
	if !ok {
		existing = set.NewSet[string]()
		s.resources[parent] = existing
	}

	existing.Add(id)

	// Root scope will not have an ID.
	if parent == "" {
		return
	}

	s.dag.AddEdge(parent, id)
}

func (s *Scope) Component(id string) (any, bool) {
	comp, ok := s.components[id]
	return comp, ok
}

func (s *Scope) NewIterator() *dag.Iterator {
	return dag.InitIterator(s.dag)
}

func (s *Scope) Resources(buildID string) []string {
	if resources, ok := s.resources[buildID]; ok {
		return resources.Values()
	}

	return nil
}

func (s *Scope) Builds(buildID string) []string {
	if builds, ok := s.builds[buildID]; ok {
		return builds.Values()
	}

	return nil
}

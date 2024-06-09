package state

import (
	"sync"
)

type Global struct {
	sync.Mutex

	target  *State
	current *State
}

func NewGlobal(target, current *State) *Global {
	return &Global{
		target:  target,
		current: current,
	}
}

func (g *Global) Target() *State {
	g.Lock()
	defer g.Unlock()

	return g.target
}

type State struct {
	sync.Mutex

	Resources map[string]*ResourceState
	Builds    map[string]*BuildState
}

func (s *State) ResourceState(id string) (*ResourceState, bool) {
	s.Lock()
	defer s.Unlock()

	r, ok := s.Resources[id]
	return r, ok
}

func (s *State) BuildState(id string) (*BuildState, bool) {
	s.Lock()
	defer s.Unlock()

	b, ok := s.Builds[id]
	return b, ok
}

type ResourceState struct {
	sync.Mutex

	Status   string
	Resource Resource
	Error    error
}

func (r *ResourceState) GetResource() Resource {
	r.Lock()
	defer r.Unlock()

	return r.Resource
}

func (r *ResourceState) GetStatus() string {
	r.Lock()
	defer r.Unlock()

	return r.Status
}

func (r *ResourceState) ToError(err error) {
	r.Lock()
	defer r.Unlock()

	r.Status = "error"
	r.Error = err
}

func (r *ResourceState) ToDone(resource Resource) {
	r.Lock()
	defer r.Unlock()

	r.Status = "done"
	r.Resource = resource
}

func (r *ResourceState) ToEvaluating() {
	r.Lock()
	defer r.Unlock()

	r.Status = "evaluating"
}

type Resource struct {
	Name       string
	Provider   Provider
	Exists     bool
	Identifier any
	Config     any
}

type Provider struct {
	Name    string
	Version string
}

type BuildState struct {
	sync.Mutex

	Status string
	Build  Build
	Error  error
}

func (b *BuildState) GetBuild() Build {
	b.Lock()
	defer b.Unlock()

	return b.Build
}

func (b *BuildState) GetStatus() string {
	b.Lock()
	defer b.Unlock()

	return b.Status
}

func (b *BuildState) ToError(err error) {
	b.Lock()
	defer b.Unlock()

	b.Status = "error"
	b.Error = err
}

func (b *BuildState) ToDone() {
	b.Lock()
	defer b.Unlock()

	b.Status = "done"
}

func (b *BuildState) ToEvaluating() {
	b.Lock()
	defer b.Unlock()

	b.Status = "evaluating"
}

type Build struct {
	Name string
}

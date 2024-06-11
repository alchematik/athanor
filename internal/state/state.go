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

type EvalState struct {
	State string
	Error error
}

type ComponentAction string

const (
	ComponentActionEmpty   = ""
	ComponentActionCreate  = "create"
	ComponentActionDelete  = "delete"
	ComponentActionUpdate  = "update"
	ComponentActionUnknown = "unknown"
)

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

	ComponentAction ComponentAction
	EvalState       EvalState
	Resource        Resource
	Error           error
}

func (r *ResourceState) GetResource() Resource {
	r.Lock()
	defer r.Unlock()

	return r.Resource
}

func (r *ResourceState) GetEvalState() EvalState {
	r.Lock()
	defer r.Unlock()

	return r.EvalState
}

func (r *ResourceState) GetComponentAction() ComponentAction {
	r.Lock()
	defer r.Unlock()

	return r.ComponentAction
}

func (r *ResourceState) SetComponentAction(a ComponentAction) {
	r.Lock()
	defer r.Unlock()

	r.ComponentAction = a
}

func (r *ResourceState) ToError(err error) {
	r.Lock()
	defer r.Unlock()

	r.EvalState.State = "error"
	r.EvalState.Error = err
}

func (r *ResourceState) ToDone(resource Resource) {
	r.Lock()
	defer r.Unlock()

	r.EvalState.State = "done"
	r.Resource = resource
}

func (r *ResourceState) ToEvaluating() {
	r.Lock()
	defer r.Unlock()

	r.EvalState.State = "evaluating"
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

	ComponentAction ComponentAction
	EvalState       EvalState
	Build           Build
	Error           error
}

func (b *BuildState) GetBuild() Build {
	b.Lock()
	defer b.Unlock()

	return b.Build
}

func (b *BuildState) GetEvalState() EvalState {
	b.Lock()
	defer b.Unlock()

	return b.EvalState
}

func (b *BuildState) GetComponentAction() ComponentAction {
	b.Lock()
	defer b.Unlock()

	return b.ComponentAction
}

func (b *BuildState) SetComponentAction(a ComponentAction) {
	b.Lock()
	defer b.Unlock()

	b.ComponentAction = a
}

func (b *BuildState) ToError(err error) {
	b.Lock()
	defer b.Unlock()

	b.EvalState.State = "error"
	b.EvalState.Error = err
}

func (b *BuildState) ToDone() {
	b.Lock()
	defer b.Unlock()

	b.EvalState.State = "done"
}

func (b *BuildState) ToEvaluating() {
	b.Lock()
	defer b.Unlock()

	b.EvalState.State = "evaluating"
}

type Build struct {
	Name string
}

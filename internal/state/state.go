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

	Name      string
	Exists    Maybe[bool]
	Resource  Maybe[Resource]
	EvalState EvalState
	Error     error
}

func (r *ResourceState) GetExists() Maybe[bool] {
	r.Lock()
	defer r.Unlock()

	return r.Exists
}

func (r *ResourceState) GetResource() Maybe[Resource] {
	r.Lock()
	defer r.Unlock()

	return r.Resource
}

func (r *ResourceState) GetEvalState() EvalState {
	r.Lock()
	defer r.Unlock()

	return r.EvalState
}

func (r *ResourceState) ToError(err error) {
	r.Lock()
	defer r.Unlock()

	r.EvalState.State = "error"
	r.EvalState.Error = err
}

func (r *ResourceState) ToDone(resource Maybe[Resource], exists Maybe[bool]) {
	r.Lock()
	defer r.Unlock()

	r.EvalState.State = "done"
	r.Resource = resource
	r.Exists = exists
}

func (r *ResourceState) ToEvaluating() {
	r.Lock()
	defer r.Unlock()

	r.EvalState.State = "evaluating"
}

type Resource struct {
	Type       Maybe[string]
	Provider   Maybe[Provider]
	Identifier Maybe[any]
	Config     Maybe[any]
	Attributes Maybe[any]
}

type Provider struct {
	Name    Maybe[string]
	Version Maybe[string]
}

type BuildState struct {
	sync.Mutex

	Name      string
	Exists    Maybe[bool]
	EvalState EvalState
	Build     Build
	Error     error
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

func (b *BuildState) GetExists() Maybe[bool] {
	b.Lock()
	defer b.Unlock()

	return b.Exists
}

func (b *BuildState) SetExists(exists Maybe[bool]) {
	b.Lock()
	defer b.Unlock()

	b.Exists = exists
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
}

type Maybe[T any] struct {
	Value   T
	Unknown bool
}

func (m Maybe[T]) Unwrap() (T, bool) {
	var value T
	if m.Unknown {
		return value, false
	}

	return m.Value, true
}

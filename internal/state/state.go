package state

import (
	"sync"
)

type State struct {
	sync.Mutex

	Resources map[string]*ResourceState
	Builds    map[string]*BuildState
}

func (s *State) Resource(id string) (*ResourceState, bool) {
	s.Lock()
	defer s.Unlock()

	r, ok := s.Resources[id]
	return r, ok
}

func (s *State) Build(id string) (*BuildState, bool) {
	s.Lock()
	defer s.Unlock()

	b, ok := s.Builds[id]
	return b, ok
}

type EvalState struct {
	State string
	Error error
}

type ResourceState struct {
	sync.Mutex

	name      string
	evalState EvalState
	error     error
	exists    bool
	resource  Resource
}

func (r *ResourceState) GetName() string {
	r.Lock()
	defer r.Unlock()

	return r.name
}

func (r *ResourceState) GetExists() bool {
	r.Lock()
	defer r.Unlock()

	return r.exists
}

func (r *ResourceState) GetResource() Resource {
	r.Lock()
	defer r.Unlock()

	return r.resource
}

func (r *ResourceState) GetEvalState() EvalState {
	r.Lock()
	defer r.Unlock()

	return r.evalState
}

func (r *ResourceState) ToError(err error) {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "error"
	r.evalState.Error = err
}

func (r *ResourceState) ToDone(resource Resource, exists bool) {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "done"
	r.resource = resource
	r.exists = exists
}

func (r *ResourceState) ToEvaluating() {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "evaluating"
}

type BuildState struct {
	sync.Mutex

	name      string
	exists    bool
	evalState EvalState
	error     error
}

func (b *BuildState) GetName() string {
	b.Lock()
	defer b.Unlock()

	return b.name
}

func (b *BuildState) GetEvalState() EvalState {
	b.Lock()
	defer b.Unlock()

	return b.evalState
}

func (b *BuildState) GetExists() bool {
	b.Lock()
	defer b.Unlock()

	return b.exists
}

func (b *BuildState) SetExists(exists bool) {
	b.Lock()
	defer b.Unlock()

	b.exists = exists
}

func (b *BuildState) ToError(err error) {
	b.Lock()
	defer b.Unlock()

	b.evalState.State = "error"
	b.evalState.Error = err
}

func (b *BuildState) ToDone() {
	b.Lock()
	defer b.Unlock()

	b.evalState.State = "done"
}

func (b *BuildState) ToEvaluating() {
	b.Lock()
	defer b.Unlock()

	b.evalState.State = "evaluating"
}

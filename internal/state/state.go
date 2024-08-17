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

func NewResourceState(name string) *ResourceState {
	return &ResourceState{name: name}
}

type ResourceState struct {
	sync.Mutex

	name         string
	evalState    EvalState
	exists       bool
	resourceType string
	provider     Provider
	identifier   any
	config       any
	attributes   any
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

func (r *ResourceState) SetExists(exists bool) {
	r.Lock()
	defer r.Unlock()

	r.exists = exists
}

func (r *ResourceState) Type() string {
	r.Lock()
	defer r.Unlock()

	return r.resourceType
}

func (r *ResourceState) SetType(t string) {
	r.Lock()
	defer r.Unlock()

	r.resourceType = t
}

func (r *ResourceState) Provider() Provider {
	r.Lock()
	defer r.Unlock()

	return r.provider
}

func (r *ResourceState) SetProvider(p Provider) {
	r.Lock()
	defer r.Unlock()

	r.provider = p
}

func (r *ResourceState) Identifier() any {
	r.Lock()
	defer r.Unlock()

	return r.identifier
}

func (r *ResourceState) SetIdentifier(id any) {
	r.Lock()
	defer r.Unlock()

	r.identifier = id
}

func (r *ResourceState) Config() any {
	r.Lock()
	defer r.Unlock()

	return r.config
}

func (r *ResourceState) SetConfig(config any) {
	r.Lock()
	defer r.Unlock()

	r.config = config
}

func (r *ResourceState) Attributes() any {
	r.Lock()
	defer r.Unlock()

	return r.attributes
}

func (r *ResourceState) SetAttributes(attrs any) {
	r.Lock()
	defer r.Unlock()

	r.attributes = attrs
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

func (r *ResourceState) ToDone() {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "done"
}

func (r *ResourceState) ToEvaluating() {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "evaluating"
}

func NewBuildState(name string) *BuildState {
	return &BuildState{name: name}
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

package plan

import (
	"sync"
)

type Plan struct {
	sync.Mutex

	Resources map[string]*ResourcePlan
	Builds    map[string]*BuildPlan
}

func (p *Plan) Resource(id string) (*ResourcePlan, bool) {
	p.Lock()
	defer p.Unlock()

	r, ok := p.Resources[id]
	return r, ok
}

func (p *Plan) Build(id string) (*BuildPlan, bool) {
	p.Lock()
	defer p.Unlock()

	b, ok := p.Builds[id]
	return b, ok
}

func NewResourcePlan(name string) *ResourcePlan {
	return &ResourcePlan{name: name}
}

type ResourcePlan struct {
	sync.Mutex

	name      string
	evalState EvalState
	error     error

	resourceType Maybe[string]
	exists       Maybe[bool]
	provider     Maybe[Provider]
	identifier   Maybe[any]
	config       Maybe[any]
}

type EvalState struct {
	State string
	Error error
}

func (r *ResourcePlan) GetName() string {
	r.Lock()
	defer r.Unlock()

	return r.name
}

func (r *ResourcePlan) GetExists() Maybe[bool] {
	r.Lock()
	defer r.Unlock()

	return r.exists
}

func (r *ResourcePlan) SetExists(exists Maybe[bool]) {
	r.Lock()
	defer r.Unlock()

	r.exists = exists
}

func (r *ResourcePlan) Provider() Maybe[Provider] {
	r.Lock()
	defer r.Unlock()

	return r.provider
}

func (r *ResourcePlan) SetProvider(provider Maybe[Provider]) {
	r.Lock()
	defer r.Unlock()

	r.provider = provider
}

func (r *ResourcePlan) Identifier() Maybe[any] {
	r.Lock()
	defer r.Unlock()

	return r.identifier
}

func (r *ResourcePlan) SetIdentifier(id Maybe[any]) {
	r.Lock()
	defer r.Unlock()

	r.identifier = id
}

func (r *ResourcePlan) Config() Maybe[any] {
	r.Lock()
	defer r.Unlock()

	return r.config
}

func (r *ResourcePlan) SetConfig(config Maybe[any]) {
	r.Lock()
	defer r.Unlock()

	r.config = config
}

func (r *ResourcePlan) Type() Maybe[string] {
	r.Lock()
	defer r.Unlock()

	return r.resourceType
}

func (r *ResourcePlan) SetType(t Maybe[string]) {
	r.Lock()
	defer r.Unlock()

	r.resourceType = t
}

func (r *ResourcePlan) GetEvalState() EvalState {
	r.Lock()
	defer r.Unlock()

	return r.evalState
}

func (r *ResourcePlan) ToError(err error) {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "error"
	r.evalState.Error = err
}

func (r *ResourcePlan) ToDone() {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "done"
}

func (r *ResourcePlan) ToEvaluating() {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "evaluating"
}

func NewBuildPlan(name string) *BuildPlan {
	return &BuildPlan{name: name}
}

type BuildPlan struct {
	sync.Mutex

	name      string
	exists    Maybe[bool]
	evalState EvalState
	error     error
}

func (b *BuildPlan) GetName() string {
	b.Lock()
	defer b.Unlock()

	return b.name
}

func (b *BuildPlan) GetEvalState() EvalState {
	b.Lock()
	defer b.Unlock()

	return b.evalState
}

func (b *BuildPlan) GetExists() Maybe[bool] {
	b.Lock()
	defer b.Unlock()

	return b.exists
}

func (b *BuildPlan) SetExists(exists Maybe[bool]) {
	b.Lock()
	defer b.Unlock()

	b.exists = exists
}

func (b *BuildPlan) ToError(err error) {
	b.Lock()
	defer b.Unlock()

	b.evalState.State = "error"
	b.evalState.Error = err
}

func (b *BuildPlan) ToDone() {
	b.Lock()
	defer b.Unlock()

	b.evalState.State = "done"
}

func (b *BuildPlan) ToEvaluating() {
	b.Lock()
	defer b.Unlock()

	b.evalState.State = "evaluating"
}

type Resource struct {
	Type       Maybe[string]
	Provider   Maybe[Provider]
	Identifier Maybe[any]
	Config     Maybe[any]
	// TODO: remove?
	Attrs Maybe[any]
}

type Provider struct {
	Name    Maybe[string]
	Version Maybe[string]
}

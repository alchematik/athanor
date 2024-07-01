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

type ResourcePlan struct {
	sync.Mutex

	name      string
	evalState EvalState
	error     error
	exists    Maybe[bool]
	resource  Maybe[Resource]
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

func (r *ResourcePlan) GetResource() Maybe[Resource] {
	r.Lock()
	defer r.Unlock()

	return r.resource
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

func (r *ResourcePlan) ToDone(resource Maybe[Resource], exists Maybe[bool]) {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "done"
	r.resource = resource
	r.exists = exists
}

func (r *ResourcePlan) ToEvaluating() {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "evaluating"
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
	Attrs      Maybe[any]
}

type Provider struct {
	Name    Maybe[string]
	Version Maybe[string]
}

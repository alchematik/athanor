package diff

import (
	"sync"

	"github.com/alchematik/athanor/internal/plan"
	"github.com/alchematik/athanor/internal/state"
)

type DiffResult struct {
	sync.Mutex

	Plan      *plan.Plan
	State     *state.State
	Resources map[string]*ResourceDiff
	Builds    map[string]*BuildDiff
}

func (d *DiffResult) Resource(id string) (*ResourceDiff, bool) {
	d.Lock()
	defer d.Unlock()

	r, ok := d.Resources[id]
	return r, ok
}

func (d *DiffResult) Build(id string) (*BuildDiff, bool) {
	d.Lock()
	defer d.Unlock()

	b, ok := d.Builds[id]
	return b, ok
}

type EvalState struct {
	State string
	Error error
}

type BuildDiff struct {
	sync.Mutex

	action    Action
	name      string
	evalState EvalState
}

func (b *BuildDiff) ToDone() {
	b.Lock()
	defer b.Unlock()

	b.evalState.State = "done"
}

func (b *BuildDiff) ToEvaluating() {
	b.Lock()
	defer b.Unlock()

	b.evalState.State = "evaluating"
}

func (b *BuildDiff) GetEvalState() EvalState {
	b.Lock()
	defer b.Unlock()

	return b.evalState
}

func (b *BuildDiff) GetAction() Action {
	b.Lock()
	defer b.Unlock()

	return b.action
}

func (b *BuildDiff) GetName() string {
	b.Lock()
	defer b.Unlock()

	return b.name
}

type ResourceDiff struct {
	sync.Mutex

	name      string
	evalState EvalState
	exists    Diff[Literal[bool]]
	resource  Diff[Resource]
}

func (r *ResourceDiff) ToEvaluating() {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "evaluating"
}

func (r *ResourceDiff) ToDone(rd Diff[Resource], exists Diff[Literal[bool]]) {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "done"
	r.resource = rd
	r.exists = exists

}

func (r *ResourceDiff) ToError(err error) {
	r.Lock()
	defer r.Unlock()

	r.evalState.State = "error"
	r.evalState.Error = err
}

func (r *ResourceDiff) GetEvalState() EvalState {
	r.Lock()
	defer r.Unlock()

	return r.evalState
}

func (r *ResourceDiff) GetName() string {
	r.Lock()
	defer r.Unlock()

	return r.name
}

func (r *ResourceDiff) GetResource() Diff[Resource] {
	r.Lock()
	defer r.Unlock()

	return r.resource
}

type Resource struct {
	Type       string
	Provider   Provider
	Identifier any
	Config     Diff[any]
}

type Provider struct {
	Name    string
	Version string
}

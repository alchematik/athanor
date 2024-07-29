package diff

import (
	"sync"

	"github.com/alchematik/athanor/internal/plan"
	"github.com/alchematik/athanor/internal/state"
)

type Diff struct {
	sync.Mutex

	Plan      *plan.Plan
	State     *state.State
	Resources map[string]*ResourceDiff
	Builds    map[string]*BuildDiff
}

type EvalState struct {
	State string
	Error error
}

type BuildDiff struct {
	sync.Mutex

	name      string
	evalState EvalState
}

type ResourceDiff struct {
	sync.Mutex

	name      string
	evalState EvalState
}

type Resource struct {
	Type       string
	Provider   Provider
	Identifier any
	Config     DiffType[any]
}

type Provider struct {
	Name    string
	Version string
}

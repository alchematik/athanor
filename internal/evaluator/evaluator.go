package evaluator

import (
	"context"
	"fmt"
	"sync"

	"github.com/alchematik/athanor/internal/selector"
	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"
)

type Evaluator struct {
	ResourceAPI ResourceAPI
	Spec        spec.Spec
	Env         state.Environment

	queueLock *sync.Mutex
}

type ResourceAPI interface {
	GetResource(context.Context, state.Resource) (state.Resource, error)
}

func NewEvaluator(api ResourceAPI, s spec.Spec, env state.Environment) *Evaluator {
	e := Evaluator{
		Env:         env,
		Spec:        s,
		ResourceAPI: api,

		queueLock: &sync.Mutex{},
	}

	return &e
}

func (e *Evaluator) Eval(ctx context.Context, sel selector.Selector) (state.Type, error) {
	s, ok := selector.SelectSpec(e.Spec, sel)
	if !ok {
		return nil, fmt.Errorf("cannot find spec with selector: %v", sel)
	}

	env, ok := selector.SelectEnvironment(e.Env, sel)
	if !ok {
		return nil, fmt.Errorf("cannot find environment for selector: %v", sel)
	}

	c, ok := s.Components[sel.Name]
	if !ok {
		return nil, fmt.Errorf("cannot find component for selector: %v", sel)
	}

	switch c := c.(type) {
	case spec.ComponentResource:
		r, err := e.resource(ctx, env, sel.Name, c)
		if err != nil {
			return nil, err
		}

		e.queueLock.Lock()

		env.States[sel.Name] = r

		e.queueLock.Unlock()

		return r, nil
	case spec.ComponentBuild:
		e.queueLock.Lock()
		defer e.queueLock.Unlock()

		res, ok := env.States[sel.Name]
		if ok {
			return res, nil
		}

		env.States[sel.Name] = state.Environment{
			States:        map[string]state.Type{},
			DependencyMap: c.Spec.DependencyMap,
		}

		return env.States[sel.Name], nil
	default:
		return nil, fmt.Errorf("not able to eval type: %T", c)
	}
}

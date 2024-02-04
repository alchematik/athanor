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

func (e *Evaluator) Eval(ctx context.Context, sel selector.Selector) error {
	s, ok := selector.SelectSpec(e.Spec, sel)
	if !ok {
		return fmt.Errorf("cannot find spec with selector: %v", sel)
	}

	env, ok := selector.SelectEnvironment(e.Env, sel)
	if !ok {
		return fmt.Errorf("cannot find environment for selector: %v", sel)
	}

	c, ok := s.Components[sel.Name]
	if !ok {
		return fmt.Errorf("cannot find component for selector: %v", sel)
	}

	switch c := c.(type) {
	case spec.ComponentResource:
		r, err := e.resource(ctx, env, sel.Name, c)
		if err != nil {
			return err
		}

		e.queueLock.Lock()

		env.States[sel.Name] = r

		e.queueLock.Unlock()
	case spec.ComponentBuild:
		e.queueLock.Lock()
		defer e.queueLock.Unlock()

		_, ok := env.States[sel.Name]
		if ok {
			return nil
		}

		env.States[sel.Name] = state.Environment{
			States:        map[string]state.Type{},
			DependencyMap: c.Spec.DependencyMap,
		}
	default:
		return fmt.Errorf("not able to eval type: %T", c)
	}

	return nil
}

func (e Evaluator) Evaluate(ctx context.Context, b spec.Spec) (state.Environment, error) {
	indegrees := map[string]int{}
	parentToChildren := map[string][]string{}
	for child, parents := range b.DependencyMap {
		indegrees[child] = len(parents)
		for _, parent := range parents {
			parentToChildren[parent] = append(parentToChildren[parent], child)
		}
	}

	// TODO: detect cycle.

	var queue []string
	for alias, in := range indegrees {
		if in == 0 {
			queue = append(queue, alias)
			delete(indegrees, alias)
		}
	}

	env := state.Environment{
		DependencyMap: b.DependencyMap,
		States:        map[string]state.Type{},
	}

	// TODO: parallelize.
	for len(queue) > 0 {
		var alias string
		alias, queue = queue[0], queue[1:]

		comp := b.Components[alias]
		if err := e.Component(ctx, env, alias, comp); err != nil {
			return state.Environment{}, err
		}

		for _, childAlias := range parentToChildren[alias] {
			indegrees[childAlias]--
			if indegrees[childAlias] == 0 {
				queue = append(queue, childAlias)
				delete(indegrees, childAlias)
			}
		}
	}

	return env, nil
}

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

	queue            []selector.Selector
	parentToChildren map[selector.Selector][]selector.Selector
	indegrees        map[selector.Selector]int
	queueLock        *sync.Mutex
}

type ResourceAPI interface {
	GetResource(context.Context, state.Resource) (state.Resource, error)
}

func NewEvaluator(api ResourceAPI, s spec.Spec, env state.Environment) *Evaluator {
	e := Evaluator{
		Env:         env,
		Spec:        s,
		ResourceAPI: api,

		parentToChildren: map[selector.Selector][]selector.Selector{},
		indegrees:        map[selector.Selector]int{},
		queueLock:        &sync.Mutex{},
		queue:            []selector.Selector{},
	}

	for alias := range s.Components {
		e.queue = append(e.queue, selector.Selector{
			Name: alias,
		})
	}

	return &e
}

func (e *Evaluator) Next() []selector.Selector {
	e.queueLock.Lock()
	defer e.queueLock.Unlock()

	out := e.queue
	e.queue = []selector.Selector{}
	return out
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

		bookend := selector.Selector{
			Name:   sel.Parent.Name,
			Parent: sel.Parent.Parent,
		}
		e.indegrees[bookend]--
		if e.indegrees[bookend] == 0 {
			e.queue = append(e.queue, bookend)
			delete(e.indegrees, bookend)
		}

		children := e.parentToChildren[sel]
		for _, child := range children {
			e.indegrees[child]--
			indegrees := e.indegrees[child]
			if indegrees == 0 {
				e.queue = append(e.queue, child)
				delete(e.indegrees, child)
			}
		}

		e.queueLock.Unlock()
	case spec.ComponentBuild:
		e.queueLock.Lock()
		defer e.queueLock.Unlock()
		st, ok := env.States[sel.Name]
		if ok {
			subEnv, ok := st.(state.Environment)
			if !ok {
				return fmt.Errorf("expected Environment, got %T", st)
			}

			subEnv.Done = true
			env.States[sel.Name] = subEnv
			return nil
		}

		env.States[sel.Name] = state.Environment{
			States: map[string]state.Type{},
		}

		e.indegrees[sel] = len(c.Spec.DependencyMap)

		for child, parents := range c.Spec.DependencyMap {
			childSelector := selector.Selector{
				Name:   child,
				Parent: &sel,
			}

			e.indegrees[childSelector] = len(parents)
			for _, parent := range parents {
				parentSelector := selector.Selector{
					Name:   parent,
					Parent: &sel,
				}
				e.parentToChildren[parentSelector] = append(e.parentToChildren[parentSelector], childSelector)
			}
		}

		for s, in := range e.indegrees {
			if in == 0 {
				e.queue = append(e.queue, s)
				delete(e.indegrees, s)
			}
		}

		// TODO: detect cycle.

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

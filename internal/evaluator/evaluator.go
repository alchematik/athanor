package evaluator

import (
	"context"
	"fmt"
	"sync"

	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"
)

type Evaluator struct {
	ResourceAPI ResourceAPI
	Spec        spec.Spec
	Env         state.Environment

	queue            []Selector
	parentToChildren map[Selector][]Selector
	indegrees        map[Selector]int
	queueLock        *sync.Mutex
}

type ResourceAPI interface {
	GetResource(context.Context, state.Resource) (state.Resource, error)
}

type Selector struct {
	Name    string
	Parent  *Selector
	Bookend bool
}

func NewEvaluator(api ResourceAPI, s spec.Spec, env state.Environment) *Evaluator {
	e := Evaluator{
		Env:         env,
		Spec:        s,
		ResourceAPI: api,

		parentToChildren: map[Selector][]Selector{},
		indegrees:        map[Selector]int{},
		queueLock:        &sync.Mutex{},
		queue:            []Selector{},
	}

	for alias := range s.Components {
		e.queue = append(e.queue, Selector{
			Name: alias,
		})
	}

	return &e
}

func (e *Evaluator) Next() []Selector {
	e.queueLock.Lock()
	defer e.queueLock.Unlock()

	out := e.queue
	e.queue = []Selector{}
	return out
}

func (e *Evaluator) Eval(ctx context.Context, selector Selector) error {
	s, ok := findSpec(e.Spec, selector)
	if !ok {
		return fmt.Errorf("cannot find spec with selector: %v", selector)
	}

	env, ok := findEnvironment(e.Env, selector)
	if !ok {
		return fmt.Errorf("cannot find environment for selector: %v", selector)
	}

	c, ok := s.Components[selector.Name]
	if !ok {
		return fmt.Errorf("cannot find component for selector: %v", selector)
	}

	switch c := c.(type) {
	case spec.ComponentResource:
		r, err := e.resource(ctx, env, selector.Name, c)
		if err != nil {
			return err
		}

		e.queueLock.Lock()

		env.States[selector.Name] = r
		bookend := Selector{
			Name:    selector.Parent.Name,
			Parent:  selector.Parent.Parent,
			Bookend: true,
		}
		e.indegrees[bookend]--
		if e.indegrees[bookend] == 0 {
			e.queue = append(e.queue, bookend)
			delete(e.indegrees, bookend)
		}

		children := e.parentToChildren[selector]
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
		if selector.Bookend {
			e.queueLock.Lock()
			st, ok := env.States[selector.Name]
			if !ok {
				return fmt.Errorf("cannot find sub state")
			}

			subEnv, ok := st.(state.Environment)
			subEnv.Done = true
			env.States[selector.Name] = subEnv
			e.queueLock.Unlock()
			return nil
		}

		e.queueLock.Lock()
		env.States[selector.Name] = state.Environment{
			States: map[string]state.Type{},
		}

		bookend := Selector{
			Name:    selector.Name,
			Parent:  selector.Parent,
			Bookend: true,
		}
		e.indegrees[bookend] = len(c.Spec.DependencyMap)

		for child, parents := range c.Spec.DependencyMap {
			childSelector := Selector{
				Name:   child,
				Parent: &selector,
			}

			e.indegrees[childSelector] = len(parents)
			for _, parent := range parents {
				parentSelector := Selector{
					Name:   parent,
					Parent: &selector,
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

		e.queueLock.Unlock()

		// TODO: detect cycle.

	default:
		return fmt.Errorf("not able to eval type: %T", c)
	}

	return nil
}

func findEnvironment(env state.Environment, selector Selector) (state.Environment, bool) {
	if selector.Parent == nil {
		return env, true
	}

	parent, ok := findEnvironment(env, *selector.Parent)
	if !ok {
		return state.Environment{}, false
	}

	st, ok := parent.States[selector.Parent.Name]
	if !ok {
		return state.Environment{}, false
	}

	envSt, ok := st.(state.Environment)

	return envSt, true
}

func findSpec(s spec.Spec, selector Selector) (spec.Spec, bool) {
	if selector.Parent == nil {
		return s, true
	}

	parent, ok := findSpec(s, *selector.Parent)
	if !ok {
		return spec.Spec{}, false
	}

	comp, ok := parent.Components[selector.Parent.Name]
	if !ok {
		return spec.Spec{}, false
	}

	build, ok := comp.(spec.ComponentBuild)
	if !ok {
		return spec.Spec{}, false
	}

	return build.Spec, true
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

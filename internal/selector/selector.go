package selector

import (
	// "fmt"
	"sync"

	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"
)

type Queuer struct {
	queue                  []Selector
	lock                   *sync.Mutex
	dependencyToDependents map[Selector][]Selector
	indegrees              map[Selector]int
	isSpec                 map[Selector]bool
}

func NewQueuer(name string, s spec.Spec) *Queuer {
	q := &Queuer{
		lock:                   &sync.Mutex{},
		dependencyToDependents: map[Selector][]Selector{},
		indegrees:              map[Selector]int{},
		isSpec:                 map[Selector]bool{},
	}
	FromSpec(q, nil, s)
	q.queue = append(q.queue, Selector{Name: name})
	return q
}

func FromSpec(q *Queuer, sel *Selector, s spec.Spec) {
	for dependent, dependencies := range s.DependencyMap {
		dependentSel := Selector{
			Name:   dependent,
			Parent: sel,
		}

		if sel != nil {
			q.dependencyToDependents[*sel] = append(q.dependencyToDependents[*sel], dependentSel)
			q.indegrees[dependentSel]++
		}

		for _, d := range dependencies {
			dependencySel := Selector{
				Name:   d,
				Parent: sel,
			}
			q.dependencyToDependents[dependencySel] = append(q.dependencyToDependents[dependencySel], dependentSel)
		}

		if c, ok := s.Components[dependent].(spec.ComponentBuild); ok {
			FromSpec(q, &dependentSel, c.Spec)
			q.isSpec[dependentSel] = true
		}
	}

	if sel != nil {
		q.indegrees[*sel] = len(s.DependencyMap)
	}
}

func (q *Queuer) Next() []Selector {
	q.lock.Lock()
	defer q.lock.Unlock()

	// for k, v := range q.dependencyToDependents {
	// 	fmt.Printf("%v -> ", k)
	// 	for _, val := range v {
	// 		fmt.Printf("%v, ", val)
	// 	}
	// 	fmt.Println()
	// }

	queue := q.queue
	q.queue = []Selector{}
	return queue
}

func (q *Queuer) Done(s Selector) {
	q.lock.Lock()
	defer q.lock.Unlock()

	dependents := q.dependencyToDependents[s]

	// Add back to be called again.
	if q.isSpec[s] {
		q.indegrees[s] = len(dependents)
		for _, dependent := range dependents {
			q.dependencyToDependents[dependent] = append(q.dependencyToDependents[dependent], s)
		}
	}

	// fmt.Printf("DONE: %v -> %v\n", s, dependents)
	for _, dependent := range dependents {
		// fmt.Printf(">>>>>> %v\n", q.indegrees)
		q.indegrees[dependent]--
		if q.indegrees[dependent] == 0 {
			q.queue = append(q.queue, dependent)
			delete(q.indegrees, dependent)
		}
	}
}

type Selector struct {
	Name   string
	Parent *Selector
}

func SelectEnvironment(env state.Environment, selector Selector) (state.Environment, bool) {
	if selector.Parent == nil {
		return env, true
	}

	parent, ok := SelectEnvironment(env, *selector.Parent)
	if !ok {
		return state.Environment{}, false
	}

	st, ok := parent.States[selector.Parent.Name]
	if !ok {
		return state.Environment{}, false
	}

	envSt, ok := st.(state.Environment)
	if !ok {
		return state.Environment{}, false
	}

	return envSt, true
}

func SelectSpec(s spec.Spec, selector Selector) (spec.Spec, bool) {
	if selector.Parent == nil {
		return s, true
	}

	parent, ok := SelectSpec(s, *selector.Parent)
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

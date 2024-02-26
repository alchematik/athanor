package selector

import (
	"context"
	"fmt"
	"sync"

	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/differ"
	"github.com/alchematik/athanor/internal/evaluator"
	"github.com/alchematik/athanor/internal/reconcile"
	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"
	"github.com/hashicorp/go-hclog"
)

type TreeNodeStatus string

const (
	TreeNodeStatusLoading TreeNodeStatus = "loading"
	TreeNodeStatusUpdate                 = "update"
	TreeNodeStatusCreate                 = "create"
	TreeNodeStatusDelete                 = "delete"
	TreeNodeStatusDone                   = "done"
	TreeNodeStatusUnknown                = "unknown"
	TreeNodeStatusEmpty                  = ""
)

type DiffController struct {
	sync.Mutex

	TargetEvaluator *evaluator.Evaluator
	ActualEvaluator *evaluator.Evaluator
	Differ          differ.Differ
	Spec            spec.ComponentBuild
	TargetEnv       state.Environment
	ActualEnv       state.Environment
	Diff            diff.Environment

	indegrees               map[Selector]int
	dependentToDependencies map[Selector][]Selector
	logger                  hclog.Logger
}

func NewDiffController(logger hclog.Logger, s spec.ComponentBuild, target, actual *evaluator.Evaluator, d differ.Differ) *DiffController {
	c := &DiffController{
		ActualEvaluator: actual,
		ActualEnv: state.Environment{
			States: map[string]state.Type{},
		},
		TargetEvaluator: target,
		TargetEnv: state.Environment{
			States: map[string]state.Type{},
		},
		Spec: s,
		Diff: diff.Environment{
			Diffs: map[string]diff.Type{},
		},
		Differ:                  d,
		indegrees:               map[Selector]int{},
		logger:                  logger,
		dependentToDependencies: map[Selector][]Selector{},
	}

	for alias := range s.Spec.Components {
		c.Add(Selector{Name: alias})
	}

	return c
}

func (q *DiffController) Process(ctx context.Context, sel Selector) (TreeNodeStatus, error) {
	comp, ok := SelectComponent(q.Spec, sel)
	if !ok {
		return "", fmt.Errorf("component not found: %s", sel.Name)
	}

	targetEnv, ok := SelectEnvironment(q.TargetEnv, sel)
	if !ok {
		return "", fmt.Errorf("environment for selector %s not found", sel.Name)
	}

	target, err := q.TargetEvaluator.Eval(ctx, targetEnv, sel.Name, comp)
	if err != nil {
		return "", err
	}

	actualEnv, ok := SelectEnvironment(q.ActualEnv, sel)
	if !ok {
		return "", fmt.Errorf("environment for selector %s not found", sel.Name)
	}

	actual, err := q.ActualEvaluator.Eval(ctx, actualEnv, sel.Name, comp)
	if err != nil {
		return "", err
	}

	diffEnv, ok := SelectDiffEnvironment(q.Diff, sel)
	if !ok {
		return "", fmt.Errorf("diff not found: %s", sel.Name)
	}

	d, err := q.Differ.Diff(ctx, diffEnv, sel.Name, target, actual)
	if err != nil {
		return "", err
	}

	build, isBuild := comp.(spec.ComponentBuild)

	if d.Operation() == diff.OperationEmpty && isBuild {
		for dependant, dependencies := range build.Spec.DependencyMap {
			dependenciesSels := make([]Selector, len(dependencies))
			for i, d := range dependencies {
				dependenciesSels[i] = Selector{Name: d, Parent: &sel}
			}

			dependantSel := Selector{Name: dependant, Parent: &sel}
			q.Add(dependantSel, dependenciesSels...)
		}

		// Add spec as child of dependants so that the spec gets processed again when children are all done being processed.
		for child := range build.Spec.Components {
			q.Add(sel, Selector{Name: child, Parent: &sel})
		}

		return TreeNodeStatusLoading, nil
	}

	if err := q.Done(sel); err != nil {
		return "", err
	}

	switch d.Operation() {
	case diff.OperationNoop:
		return TreeNodeStatusEmpty, nil
	case diff.OperationCreate:
		return TreeNodeStatusCreate, nil
	case diff.OperationUpdate:
		return TreeNodeStatusUpdate, nil
	case diff.OperationDelete:
		return TreeNodeStatusDelete, nil
	case diff.OperationUnknown:
		return TreeNodeStatusUnknown, nil
	default:
		return "", fmt.Errorf("unhandled diff operation: %s", d.Operation())
	}
}

func (q *DiffController) Add(s Selector, dependencies ...Selector) {
	q.Lock()
	defer q.Unlock()

	for _, d := range dependencies {
		q.dependentToDependencies[d] = append(q.dependentToDependencies[d], s)
	}

	q.indegrees[s] += len(dependencies)
}

func (q *DiffController) Done(s Selector) error {
	q.Lock()
	defer q.Unlock()

	dependents := q.dependentToDependencies[s]
	for _, d := range dependents {
		q.indegrees[d]--
	}

	return nil
}

func (q *DiffController) Next() []Selector {
	q.Lock()
	defer q.Unlock()

	var queue []Selector
	for sel, indegrees := range q.indegrees {
		if indegrees == 0 {
			queue = append(queue, sel)
			delete(q.indegrees, sel)
		}
	}

	return queue
}

type ReconcileController struct {
	sync.Mutex

	Spec       spec.ComponentBuild
	Result     state.Environment
	Diff       diff.Environment
	Reconciler *reconcile.Reconciler

	indegrees               map[Selector]int
	dependentToDependencies map[Selector][]Selector
	logger                  hclog.Logger
}

func NewReconcileController(l hclog.Logger, s spec.ComponentBuild, d diff.Environment, r *reconcile.Reconciler) *ReconcileController {
	c := &ReconcileController{
		Spec: s,
		Result: state.Environment{
			States: map[string]state.Type{},
		},
		Diff:                    d,
		Reconciler:              r,
		indegrees:               map[Selector]int{},
		dependentToDependencies: map[Selector][]Selector{},
		logger:                  l,
	}

	for alias := range s.Spec.Components {
		c.Add(Selector{Name: alias})
	}

	return c
}

func (r *ReconcileController) Process(ctx context.Context, sel Selector) (TreeNodeStatus, error) {
	env, ok := SelectEnvironment(r.Result, sel)
	if !ok {
		return "", fmt.Errorf("cannot find parent environment: %s", sel.Name)
	}

	diffEnv, ok := SelectDiffEnvironment(r.Diff, sel)
	if !ok {
		return "", fmt.Errorf("cannot find diff environment: %s", sel.Name)
	}

	d, ok := diffEnv.Diffs[sel.Name]
	if !ok {
		return "", fmt.Errorf("cannot find diff: %s", sel.Name)
	}

	if d.Operation() == diff.OperationNoop {
		return TreeNodeStatusEmpty, nil
	}

	_, err := r.Reconciler.Reconcile(ctx, env, sel.Name, d)
	if err != nil {
		return "", err
	}

	comp, ok := SelectComponent(r.Spec, sel)
	if !ok {
		return "", fmt.Errorf("cannot find component: %s", sel.Name)
	}

	build, isBuild := comp.(spec.ComponentBuild)
	if isBuild {
		st, ok := env.States[sel.Name].(state.Environment)
		if !ok {
			return "", fmt.Errorf("expected environment: %s", sel.Name)
		}

		if len(st.States) == len(build.Spec.Components) {
			return TreeNodeStatusDone, nil
		}

		current, ok := d.(diff.Environment)
		if !ok {
			return "", fmt.Errorf("expected %s to be environment", sel.Name)
		}

		dependentToDependencies := map[Selector][]Selector{}
		for dependent, dependencies := range build.Spec.DependencyMap {
			dependentSel := Selector{Name: dependent, Parent: &sel}
			for _, dependency := range dependencies {
				dependencySel := Selector{Name: dependency, Parent: &sel}
				dependentToDependencies[dependencySel] = append(dependentToDependencies[dependencySel], dependentSel)
			}
		}

		for alias, dif := range current.Diffs {
			s := Selector{Name: alias, Parent: &sel}
			if dif.Operation() == diff.OperationDelete {
				r.Add(s, dependentToDependencies[s]...)
			} else {
				list := make([]Selector, len(build.Spec.DependencyMap[alias]))
				for i, dep := range build.Spec.DependencyMap[alias] {
					list[i] = Selector{Name: dep, Parent: &sel}
				}

				r.Add(s, list...)
			}
		}

		for childAlias := range build.Spec.Components {
			r.Add(sel, Selector{Name: childAlias, Parent: &sel})
		}
	}

	if err := r.Done(sel); err != nil {
		return "", err
	}

	return "", nil
}

func (r *ReconcileController) Add(s Selector, dependencies ...Selector) {
	r.Lock()
	defer r.Unlock()

	for _, d := range dependencies {
		r.dependentToDependencies[d] = append(r.dependentToDependencies[d], s)
	}

	r.indegrees[s] += len(dependencies)
}

func (r *ReconcileController) Done(s Selector) error {
	r.Lock()
	defer r.Unlock()

	dependents := r.dependentToDependencies[s]
	for _, d := range dependents {
		r.indegrees[d]--
	}

	return nil
}

func (r *ReconcileController) Next() []Selector {
	r.Lock()
	defer r.Unlock()

	var queue []Selector
	for sel, indegrees := range r.indegrees {
		if indegrees == 0 {
			queue = append(queue, sel)
			delete(r.indegrees, sel)
		}
	}

	return queue
}

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

func SelectState(env state.Environment, sel Selector) (state.Type, bool) {
	var parent state.Environment
	if sel.Parent == nil {
		parent = env
	} else {
		s, ok := SelectState(env, *sel.Parent)
		if !ok {
			return nil, false
		}

		e, ok := s.(state.Environment)
		if !ok {
			return nil, false
		}

		parent = e
	}

	s, ok := parent.States[sel.Name]
	return s, ok
}

func SelectComponent(s spec.ComponentBuild, sel Selector) (spec.Component, bool) {
	var parent spec.ComponentBuild
	if sel.Parent == nil {
		parent = s
	} else {
		comp, ok := SelectComponent(s, *sel.Parent)
		if !ok {
			return nil, false
		}

		build, ok := comp.(spec.ComponentBuild)
		if !ok {
			return nil, false
		}

		parent = build
	}

	c, ok := parent.Spec.Components[sel.Name]
	return c, ok
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

func SelectDiffEnvironment(env diff.Environment, selector Selector) (diff.Environment, bool) {
	if selector.Parent == nil {
		return env, true
	}

	parent, ok := SelectDiffEnvironment(env, *selector.Parent)
	if !ok {
		return diff.Environment{}, false
	}

	d, ok := parent.Diffs[selector.Parent.Name]
	if !ok {
		return diff.Environment{}, false
	}

	sub, ok := d.(diff.Environment)

	return sub, true
}

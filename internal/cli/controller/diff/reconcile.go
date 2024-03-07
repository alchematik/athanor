package diff

import (
	"context"
	"fmt"
	"sync"

	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/reconcile"
	"github.com/alchematik/athanor/internal/selector"
	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"

	"github.com/hashicorp/go-hclog"
)

type ReconcileController struct {
	sync.Mutex

	Spec       spec.ComponentBuild
	Result     state.Environment
	Diff       diff.Environment
	Reconciler *reconcile.Reconciler

	indegrees               map[selector.Selector]int
	dependentToDependencies map[selector.Selector][]selector.Selector
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
		indegrees:               map[selector.Selector]int{},
		dependentToDependencies: map[selector.Selector][]selector.Selector{},
		logger:                  l,
	}

	for alias := range s.Spec.Components {
		c.Add(selector.Selector{Name: alias})
	}

	return c
}

func (r *ReconcileController) Process(ctx context.Context, sel selector.Selector) (TreeNodeStatus, error) {
	// r.Lock()
	// for k, v := range r.indegrees {
	// 	log.Printf("%v -> \n %v\n", k.Name, v)
	// }
	// r.Unlock()
	env, ok := selector.SelectEnvironment(r.Result, sel)
	if !ok {
		return "", fmt.Errorf("cannot find parent environment: %s", sel.Name)
	}

	diffEnv, ok := selector.SelectDiffEnvironment(r.Diff, sel)
	if !ok {
		return "", fmt.Errorf("cannot find diff environment: %s", sel.Name)
	}

	d, ok := diffEnv.Diffs[sel.Name]
	if !ok {
		return "", fmt.Errorf("cannot find diff: %s", sel.Name)
	}

	// if d.Operation() == diff.OperationNoop {
	// 	if err := r.Done(sel); err != nil {
	// 		return "", err
	// 	}
	//
	// 	return TreeNodeStatusEmpty, nil
	// }

	// Finished processing environment because got called a second time.
	r.Lock()
	_, ok = env.States[sel.Name]
	r.Unlock()
	if ok && d.Operation() != diff.OperationEmpty {
		if err := r.Done(sel); err != nil {
			return "", err
		}
		return TreeNodeStatusDone, nil
	}

	_, err := r.Reconciler.Reconcile(ctx, env, sel.Name, d)
	if err != nil {
		return "", err
	}

	comp, ok := selector.SelectComponent(r.Spec, sel)
	if !ok {
		return "", fmt.Errorf("cannot find component: %s", sel.Name)
	}

	build, isBuild := comp.(spec.ComponentBuild)
	if isBuild {
		current, ok := d.(diff.Environment)
		if !ok {
			return "", fmt.Errorf("expected %s to be environment", sel.Name)
		}

		dependentToDependencies := map[selector.Selector][]selector.Selector{}
		for dependent, dependencies := range build.Spec.DependencyMap {
			dependentSel := selector.Selector{Name: dependent, Parent: &sel}
			for _, dependency := range dependencies {
				dependencySel := selector.Selector{Name: dependency, Parent: &sel}
				dependentToDependencies[dependencySel] = append(dependentToDependencies[dependencySel], dependentSel)
			}
		}

		for alias, dif := range current.Diffs {
			s := selector.Selector{Name: alias, Parent: &sel}
			if dif.Operation() == diff.OperationDelete {
				r.Add(s, dependentToDependencies[s]...)
			} else {
				list := make([]selector.Selector, len(build.Spec.DependencyMap[alias]))
				for i, dep := range build.Spec.DependencyMap[alias] {
					list[i] = selector.Selector{Name: dep, Parent: &sel}
				}

				r.Add(s, list...)
			}
		}

		for childAlias := range build.Spec.Components {
			r.Add(sel, selector.Selector{Name: childAlias, Parent: &sel})
		}

		return TreeNodeStatusLoading, nil
	}

	if err := r.Done(sel); err != nil {
		return "", err
	}

	if d.Operation() == diff.OperationNoop {
		return TreeNodeStatusEmpty, nil
	}

	return TreeNodeStatusDone, nil
}

func (r *ReconcileController) Add(s selector.Selector, dependencies ...selector.Selector) {
	r.Lock()
	defer r.Unlock()

	for _, d := range dependencies {
		r.dependentToDependencies[d] = append(r.dependentToDependencies[d], s)
	}

	r.indegrees[s] += len(dependencies)
}

func (r *ReconcileController) Done(s selector.Selector) error {
	r.Lock()
	defer r.Unlock()

	dependents := r.dependentToDependencies[s]
	for _, d := range dependents {
		r.indegrees[d]--
	}

	return nil
}

func (r *ReconcileController) Next() []selector.Selector {
	r.Lock()
	defer r.Unlock()

	var queue []selector.Selector
	for sel, indegrees := range r.indegrees {
		if indegrees == 0 {
			queue = append(queue, sel)
			delete(r.indegrees, sel)
		}
	}

	return queue
}

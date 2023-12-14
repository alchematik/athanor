package reconcile

import (
	"fmt"
	"github.com/alchematik/athanor/diff"
)

type Reconciler struct{}

func (r Reconciler) Reconcile(d diff.Type) error {
	switch dif := d.(type) {
	case diff.Environment:
		return r.ReconcileEnvironment(dif)
	case diff.Resource:
		return r.ReconcileResource(dif)
	default:
		return fmt.Errorf("unsupported type for reconciliation: %%", d)
	}
}

func (r Reconciler) ReconcileEnvironment(d diff.Environment) error {
	indegrees := map[string]int{}
	for parent, children := range d.Dependencies {
		if _, ok := indegrees[parent]; !ok {
			indegrees[parent] = 0
		}

		for _, child := range children {
			indegrees[child]++
		}
	}

	var queue []string
	for alias, degrees := range indegrees {
		if degrees == 0 {
			queue = append(queue, alias)
			delete(indegrees, alias)
		}
	}

	// TODO: parallelize.
	for len(queue) > 0 {
		var alias string
		alias, queue = queue[0], queue[1:]

		fmt.Printf("reconciling: %v\n", alias)
		if err := r.Reconcile(d.Diffs[alias]); err != nil {
			return err
		}

		for _, child := range d.Dependencies[alias] {
			indegrees[child]--
			if indegrees[child] == 0 {
				queue = append(queue, child)
				delete(indegrees, child)
			}
		}
	}

	return nil
}

func (r Reconciler) ReconcileResource(d diff.Resource) error {
	return nil
}

package diff

import (
	"context"
	"fmt"
	"sync"

	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/differ"
	"github.com/alchematik/athanor/internal/evaluator"
	"github.com/alchematik/athanor/internal/selector"
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
	Differ          *differ.Differ
	Spec            spec.ComponentBuild
	TargetEnv       state.Environment
	ActualEnv       state.Environment
	Diff            diff.Environment

	indegrees               map[selector.Selector]int
	dependentToDependencies map[selector.Selector][]selector.Selector
	logger                  hclog.Logger
}

func NewDiffController(logger hclog.Logger, s spec.ComponentBuild, target, actual *evaluator.Evaluator, d *differ.Differ) *DiffController {
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
		indegrees:               map[selector.Selector]int{},
		logger:                  logger,
		dependentToDependencies: map[selector.Selector][]selector.Selector{},
	}

	for alias := range s.Spec.Components {
		c.Add(selector.Selector{Name: alias})
	}

	return c
}

func (q *DiffController) Process(ctx context.Context, sel selector.Selector) (TreeNodeStatus, error) {
	comp, ok := selector.SelectComponent(q.Spec, sel)
	if !ok {
		return "", fmt.Errorf("component not found: %s", sel.Name)
	}

	targetEnv, ok := selector.SelectEnvironment(q.TargetEnv, sel)
	if !ok {
		return "", fmt.Errorf("environment for selector %s not found", sel.Name)
	}

	target, err := q.TargetEvaluator.Eval(ctx, targetEnv, sel.Name, comp)
	if err != nil {
		return "", err
	}

	actualEnv, ok := selector.SelectEnvironment(q.ActualEnv, sel)
	if !ok {
		return "", fmt.Errorf("environment for selector %s not found", sel.Name)
	}

	actual, err := q.ActualEvaluator.Eval(ctx, actualEnv, sel.Name, comp)
	if err != nil {
		return "", err
	}

	diffEnv, ok := selector.SelectDiffEnvironment(q.Diff, sel)
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
			dependenciesSels := make([]selector.Selector, len(dependencies))
			for i, d := range dependencies {
				dependenciesSels[i] = selector.Selector{Name: d, Parent: &sel}
			}

			dependantSel := selector.Selector{Name: dependant, Parent: &sel}
			q.Add(dependantSel, dependenciesSels...)
		}

		// Add spec as child of dependants so that the spec gets processed again when children are all done being processed.
		for child := range build.Spec.Components {
			q.Add(sel, selector.Selector{Name: child, Parent: &sel})
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

func (q *DiffController) Add(s selector.Selector, dependencies ...selector.Selector) {
	q.Lock()
	defer q.Unlock()

	for _, d := range dependencies {
		q.dependentToDependencies[d] = append(q.dependentToDependencies[d], s)
	}

	q.indegrees[s] += len(dependencies)
}

func (q *DiffController) Done(s selector.Selector) error {
	q.Lock()
	defer q.Unlock()

	dependents := q.dependentToDependencies[s]
	for _, d := range dependents {
		q.indegrees[d]--
	}

	return nil
}

func (q *DiffController) Next() []selector.Selector {
	q.Lock()
	defer q.Unlock()

	var queue []selector.Selector
	for sel, indegrees := range q.indegrees {
		if indegrees == 0 {
			queue = append(queue, sel)
			delete(q.indegrees, sel)
		}
	}

	return queue
}

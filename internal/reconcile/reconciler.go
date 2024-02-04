package reconcile

import (
	"context"
	"fmt"
	"sync"

	api "github.com/alchematik/athanor/internal/api/resource"
	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/selector"
	"github.com/alchematik/athanor/internal/state"
)

type Reconciler struct {
	ResourceAPI ResourceAPI
	Env         diff.Environment
	Result      state.Environment

	queue            []selector.Selector
	queueLock        *sync.Mutex
	parentToChildren map[selector.Selector][]selector.Selector
	indegrees        map[selector.Selector]int
}

type ResourceAPI interface {
	GetResource(context.Context, state.Resource) (state.Resource, error)
	CreateResource(context.Context, state.Resource) (state.Resource, error)
	DeleteResource(context.Context, state.Resource) error
	UpdateResource(context.Context, state.Resource, []api.Field) (state.Resource, error)
}

func NewReconciler(api ResourceAPI, env diff.Environment, result state.Environment) *Reconciler {
	var queue []selector.Selector
	for alias := range env.Diffs {
		queue = append(queue, selector.Selector{Name: alias})
	}
	return &Reconciler{
		ResourceAPI:      api,
		Env:              env,
		Result:           result,
		queueLock:        &sync.Mutex{},
		queue:            queue,
		parentToChildren: map[selector.Selector][]selector.Selector{},
		indegrees:        map[selector.Selector]int{},
	}
}

func (r *Reconciler) Next() []selector.Selector {
	r.queueLock.Lock()
	defer r.queueLock.Unlock()

	out := r.queue
	r.queue = []selector.Selector{}
	return out
}

func (r *Reconciler) Reconcile(ctx context.Context, sel selector.Selector) error {
	e, ok := diff.SelectDiffEnvironment(r.Env, sel)
	if !ok {
		return fmt.Errorf("cannot find environment with selector: %v", sel)
	}

	env, ok := selector.SelectEnvironment(r.Result, sel)
	if !ok {
		return fmt.Errorf("cannot find result environment with selector: %v", sel)
	}

	current, ok := e.Diffs[sel.Name]
	if !ok {
		return fmt.Errorf("cannot find diff with selector: %v", sel)
	}

	switch d := current.(type) {
	case diff.Resource:
		res, err := r.ReconcileResource(ctx, r.Result, d)
		if err != nil {
			return err
		}

		r.queueLock.Lock()
		env.States[sel.Name] = res
		parent := *sel.Parent
		r.indegrees[parent]--
		if r.indegrees[parent] == 0 {
			r.queue = append(r.queue, parent)
			delete(r.indegrees, parent)
		}

		children := r.parentToChildren[sel]
		for _, child := range children {
			r.indegrees[child]--
			if r.indegrees[child] == 0 {
				r.queue = append(r.queue, child)
				delete(r.indegrees, child)
			}
		}
		r.queueLock.Unlock()
	case diff.Environment:
		r.queueLock.Lock()
		defer r.queueLock.Unlock()

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

		r.indegrees[sel] = len(d.Diffs)
		for child, parents := range d.Dependencies {
			childSelector := selector.Selector{
				Name:   child,
				Parent: &sel,
			}

			r.indegrees[childSelector] = len(parents)
			for _, parent := range parents {
				parentSelector := selector.Selector{
					Name:   parent,
					Parent: &sel,
				}
				r.parentToChildren[parentSelector] = append(r.parentToChildren[parentSelector], childSelector)
			}
		}

		for s, in := range r.indegrees {
			if in == 0 {
				r.queue = append(r.queue, s)
				delete(r.indegrees, s)
			}
		}

	}

	return nil
}

func (r Reconciler) ReconcileEnvironment(ctx context.Context, d diff.Environment) (state.Environment, error) {
	indegrees := map[string]int{}
	parentToChildren := map[string][]string{}
	for child, parents := range d.Dependencies {
		indegrees[child] = len(parents)
		for _, parent := range parents {
			parentToChildren[parent] = append(parentToChildren[parent], child)
		}
	}

	var queue []string
	for alias, degrees := range indegrees {
		if degrees == 0 {
			queue = append(queue, alias)
			delete(indegrees, alias)
		}
	}

	reconciledEnv := state.Environment{
		States:        map[string]state.Type{},
		DependencyMap: d.Dependencies,
	}

	// TODO: parallelize.
	for len(queue) > 0 {
		var alias string
		alias, queue = queue[0], queue[1:]

		var reconciled state.Type
		switch current := d.Diffs[alias].(type) {
		case diff.Resource:
			resolved, err := resolve(reconciledEnv, current.To)
			if err != nil {
				return state.Environment{}, err
			}

			resolvedResource, ok := resolved.(state.Resource)
			if !ok {
				return state.Environment{}, fmt.Errorf("expected resource, got %T", resolved)
			}

			updatedDiff, err := diff.Diff(current.From, resolvedResource)
			if err != nil {
				return state.Environment{}, err
			}

			resourceDiff, ok := updatedDiff.(diff.Resource)
			if !ok {
				return state.Environment{}, fmt.Errorf("expected resource diff, got %T", updatedDiff)
			}

			reconciled, err = r.ReconcileResource(ctx, reconciledEnv, resourceDiff)
			if err != nil {
				return state.Environment{}, err
			}
		case diff.Environment:
			var err error
			reconciled, err = r.ReconcileEnvironment(ctx, current)
			if err != nil {
				return state.Environment{}, err
			}
		}

		reconciledEnv.States[alias] = reconciled

		for _, child := range parentToChildren[alias] {
			indegrees[child]--
			if indegrees[child] == 0 {
				queue = append(queue, child)
				delete(indegrees, child)
			}
		}
	}

	return reconciledEnv, nil
}

func (r Reconciler) ReconcileResource(ctx context.Context, env state.Environment, d diff.Resource) (state.Resource, error) {
	switch d.Operation() {
	case diff.OperationNoop:
		return d.To, nil
	case diff.OperationCreate:
		return r.ResourceAPI.CreateResource(ctx, d.To)
	case diff.OperationDelete:
		return d.To, r.ResourceAPI.DeleteResource(ctx, d.To)
	case diff.OperationUpdate:
		mask, err := diffToUpdateMask(d.ConfigDiff)
		if err != nil {
			return state.Resource{}, err
		}

		return r.ResourceAPI.UpdateResource(ctx, d.To, mask)
	default:
		return state.Resource{}, fmt.Errorf("unsupported operation: %v\n", d.Operation())
	}
}

func diffToUpdateMask(d diff.Type) ([]api.Field, error) {
	switch t := d.(type) {
	case diff.Resource:
		return diffToUpdateMask(t.ConfigDiff)
	case diff.Map:
		var fields []api.Field
		for k, v := range t.Diffs {
			// Skip noops .
			if v.Operation() == diff.OperationNoop {
				continue
			}

			op := api.OperationUpdate
			if v.Operation() == diff.OperationDelete {
				op = api.OperationDelete
			}

			sub, err := diffToUpdateMask(v)
			if err != nil {
				return nil, err
			}

			fields = append(fields, api.Field{Name: k, SubFields: sub, Operation: op})
		}

		return fields, nil
	case diff.String:
		return nil, nil
	case diff.File:
		return nil, nil
	case diff.Immutable:
		return nil, nil
	case diff.List:
		var fields []api.Field
		for _, d := range t.Diffs {
			if d.Operation() == diff.OperationNoop {
				continue
			}

			op := api.OperationUpdate
			if d.Operation() == diff.OperationDelete {
				op = api.OperationDelete
			}

			m, err := diffToUpdateMask(d)
			if err != nil {
				return nil, err
			}

			fields = append(fields, api.Field{SubFields: m, Operation: op})
		}

		return fields, nil
	default:
		return nil, fmt.Errorf("unsupported type for mask %T\n", d)
	}
}

func resolve(env state.Environment, res state.Type) (state.Type, error) {
	switch r := res.(type) {
	case state.String:
		return r, nil
	case state.File:
		return r, nil
	case state.Identifier:
		val, err := resolve(env, r.Value)
		if err != nil {
			return nil, err
		}

		return state.Identifier{ResourceType: r.ResourceType, Value: val, Alias: r.Alias}, nil
	case state.Map:
		m := state.Map{
			Entries: map[string]state.Type{},
		}
		for k, v := range r.Entries {
			resolved, err := resolve(env, v)
			if err != nil {
				return nil, err
			}

			m.Entries[k] = resolved
		}

		return m, nil
	case state.List:
		l := state.List{Elements: make([]state.Type, len(r.Elements))}
		for i, e := range r.Elements {
			val, err := resolve(env, e)
			if err != nil {
				return nil, err
			}

			l.Elements[i] = val
		}

		return l, nil
	case state.ResourceRef:
		res, ok := env.States[r.Alias]
		if !ok {
			return nil, fmt.Errorf("resolve: no resource with alias %q found", r.Alias)
		}

		return res, nil
	case state.Resource:
		config, err := resolve(env, r.Config)
		if err != nil {
			return nil, err
		}

		return state.Resource{
			Provider:   r.Provider,
			Identifier: r.Identifier,
			Config:     config,
			Attrs:      r.Attrs,
			Exists:     r.Exists,
		}, nil
	case state.Immutable:
		return resolve(env, r.Value)
	case state.Unknown:
		resolved, err := resolve(env, r.Object)
		if err != nil {
			return nil, err
		}

		var m map[string]state.Type
		switch obj := resolved.(type) {
		case state.Resource:
			m = map[string]state.Type{
				"identifier": obj.Identifier,
				"config":     obj.Config,
				"attrs":      obj.Attrs,
			}
		case state.Map:
			m = obj.Entries
		default:
			return nil, fmt.Errorf("value type [%T] has no field %q", resolved, r.Name)
		}

		val, ok := m[r.Name]
		if !ok {
			return nil, fmt.Errorf("property %q not set", r.Name)
		}

		return val, nil
	default:
		return nil, fmt.Errorf("invalid type to resolve: %T", res)
	}
}

package reconcile

import (
	"context"
	"fmt"
	"sync"

	api "github.com/alchematik/athanor/internal/api/resource"
	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/state"
)

type Reconciler struct {
	ResourceAPI ResourceAPI

	queueLock *sync.Mutex
}

type ResourceAPI interface {
	GetResource(context.Context, state.Resource) (state.Resource, error)
	CreateResource(context.Context, state.Resource) (state.Resource, error)
	DeleteResource(context.Context, state.Resource) error
	UpdateResource(context.Context, state.Resource, []api.Field) (state.Resource, error)
}

func NewReconciler(api ResourceAPI) *Reconciler {
	return &Reconciler{
		ResourceAPI: api,
		queueLock:   &sync.Mutex{},
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, env state.Environment, alias string, current diff.Type) (state.Type, error) {
	switch d := current.(type) {
	case diff.Resource:
		res, err := r.ReconcileResource(ctx, env, d)
		if err != nil {
			return nil, err
		}

		r.queueLock.Lock()

		env.States[alias] = res

		r.queueLock.Unlock()

		return res, nil
	case diff.Environment:
		r.queueLock.Lock()
		res, ok := env.States[alias]
		r.queueLock.Unlock()

		if ok {
			return res, nil
		}

		runtimeConfig, err := r.resolve(env, d.To.RuntimeConfig)
		if err != nil {
			return nil, err
		}

		res = state.Environment{
			States:        map[string]state.Type{},
			RuntimeConfig: runtimeConfig,
		}

		r.queueLock.Lock()
		env.States[alias] = res
		r.queueLock.Unlock()

		return res, nil
	default:
		return nil, fmt.Errorf("unhandled type while reconciling: %T\n", current)
	}
}

func (r *Reconciler) ReconcileResource(ctx context.Context, env state.Environment, d diff.Resource) (state.Resource, error) {
	if d.Operation() == diff.OperationUnknown {
		to, err := r.resolveResource(env, d.To)
		if err != nil {
			return state.Resource{}, err
		}

		updatedDiff, err := diff.Diff(d.From, to)
		if err != nil {
			return state.Resource{}, err
		}

		var ok bool
		d, ok = updatedDiff.(diff.Resource)
		if !ok {
			return state.Resource{}, fmt.Errorf("not a resource diff")
		}
	}

	switch d.Operation() {
	case diff.OperationNoop:
		return d.From, nil
	case diff.OperationCreate:
		return r.ResourceAPI.CreateResource(ctx, d.To)
	case diff.OperationDelete:
		return d.To, r.ResourceAPI.DeleteResource(ctx, d.To)
	case diff.OperationUpdate:
		to, err := r.resolveResource(env, d.To)
		if err != nil {
			return state.Resource{}, err
		}

		mask, err := diffToUpdateMask(d.ConfigDiff)
		if err != nil {
			return state.Resource{}, err
		}

		return r.ResourceAPI.UpdateResource(ctx, to, mask)
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
			// Skip noops.
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
	case diff.Unknown:
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
	case diff.Identifier:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported type for mask %T\n", d)
	}
}

func (r *Reconciler) resolveResource(env state.Environment, res state.Resource) (state.Resource, error) {
	id, err := r.resolveIdentifier(env, res.Identifier)
	if err != nil {
		return state.Resource{}, err
	}

	config, err := r.resolve(env, res.Config)
	if err != nil {
		return state.Resource{}, err
	}

	return state.Resource{
		Provider:   res.Provider,
		Identifier: id,
		Config:     config,
		Attrs:      res.Attrs,
		Exists:     res.Exists,
	}, nil
}

func (r *Reconciler) resolveIdentifier(env state.Environment, res state.Identifier) (state.Identifier, error) {
	val, err := r.resolve(env, res.Value)
	if err != nil {
		return state.Identifier{}, err
	}

	return state.Identifier{ResourceType: res.ResourceType, Value: val, Alias: res.Alias}, nil
}

func (r *Reconciler) resolve(env state.Environment, res state.Type) (state.Type, error) {
	switch res := res.(type) {
	case state.String:
		return res, nil
	case state.File:
		return res, nil
	case state.Identifier:
		return r.resolveIdentifier(env, res)
	case state.Map:
		m := state.Map{
			Entries: map[string]state.Type{},
		}
		for k, v := range res.Entries {
			resolved, err := r.resolve(env, v)
			if err != nil {
				return nil, err
			}

			m.Entries[k] = resolved
		}

		return m, nil
	case state.List:
		l := state.List{Elements: make([]state.Type, len(res.Elements))}
		for i, e := range res.Elements {
			val, err := r.resolve(env, e)
			if err != nil {
				return nil, err
			}

			l.Elements[i] = val
		}

		return l, nil
	case state.ResourceRef:
		r.queueLock.Lock()
		ref, ok := env.States[res.Alias]
		r.queueLock.Unlock()
		if !ok {
			return nil, fmt.Errorf("resolve: no resource with alias %q found", res.Alias)
		}

		return ref, nil
	case state.Resource:
		return r.resolveResource(env, res)
	case state.Immutable:
		return r.resolve(env, res.Value)
	case state.Unknown:
		resolved, err := r.resolve(env, res.Object)
		if err != nil {
			return nil, err
		}

		if res.Name == "" {
			return resolved, nil
		}

		var m map[string]state.Type
		switch obj := resolved.(type) {
		case state.Resource:
			m = map[string]state.Type{
				"identifier": obj.Identifier,
				"config":     obj.Config,
				"attrs":      obj.Attrs,
			}
		case state.Nil:
			m = env.States
		case state.Map:
			m = obj.Entries
		default:
			return nil, fmt.Errorf("value type [%T] has no field %q", resolved, res.Name)
		}

		r.queueLock.Lock()
		val, ok := m[res.Name]
		r.queueLock.Unlock()
		if !ok {
			return nil, fmt.Errorf("property %q not set", res.Name)
		}

		return val, nil
	case state.RuntimeConfig:
		return env.RuntimeConfig, nil
	case state.Nil:
		return state.Nil{}, nil
	default:
		return nil, fmt.Errorf("invalid type to resolve: %T", res)
	}
}

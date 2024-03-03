package evaluator

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"
)

type Evaluator struct {
	ResourceAPI ResourceAPI

	queueLock *sync.Mutex
}

type ResourceAPI interface {
	GetResource(context.Context, state.Resource) (state.Resource, error)
}

func NewEvaluator(api ResourceAPI) *Evaluator {
	e := Evaluator{
		ResourceAPI: api,

		queueLock: &sync.Mutex{},
	}

	return &e
}

func (e *Evaluator) Eval(ctx context.Context, env state.Environment, alias string, c spec.Component) (state.Type, error) {
	switch c := c.(type) {
	case spec.ComponentResource:
		r, err := e.resource(ctx, env, c)
		if err != nil {
			return nil, err
		}

		e.queueLock.Lock()

		env.States[alias] = r

		e.queueLock.Unlock()

		return r, nil
	case spec.ComponentBuild:
		e.queueLock.Lock()
		defer e.queueLock.Unlock()

		res, ok := env.States[alias]
		if ok {
			return res, nil
		}

		runtimeConfig, err := e.Value(ctx, env, c.Spec.RuntimeConfig)
		if err != nil {
			return nil, err
		}

		log.Printf("RUNTIME CONFIG >> %+v\n", runtimeConfig)

		env.States[alias] = state.Environment{
			States:        map[string]state.Type{},
			RuntimeConfig: runtimeConfig,
		}

		return env.States[alias], nil
	default:
		return nil, fmt.Errorf("not able to eval type: %T", c)
	}
}

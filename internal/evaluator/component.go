package evaluator

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"
)

func (e Evaluator) Component(ctx context.Context, env state.Environment, alias string, comp spec.Component) error {
	switch c := comp.(type) {
	case spec.ComponentResource:
		return e.resource(ctx, env, alias, c)
	case spec.ComponentBuild:
		return e.build(ctx, env, alias, c)
	default:
		return fmt.Errorf("unknown component type: %T", comp)
	}
}

func (e Evaluator) resource(ctx context.Context, env state.Environment, alias string, comp spec.ComponentResource) error {
	val, err := e.Value(ctx, env, comp.Value)
	if err != nil {
		return err
	}

	r, ok := val.(state.Resource)
	if !ok {
		return fmt.Errorf("expected Resource type, got %T", val)
	}

	env.States[alias] = r
	return nil
}

func (e Evaluator) build(ctx context.Context, env state.Environment, alias string, comp spec.ComponentBuild) error {
	subEnv, err := e.Evaluate(ctx, comp.Spec)
	if err != nil {
		return err
	}

	env.States[alias] = subEnv

	return nil
}

package evaluator

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"
)

func (e Evaluator) resource(ctx context.Context, env state.Environment, alias string, comp spec.ComponentResource) (state.Resource, error) {
	val, err := e.Value(ctx, env, comp.Value)
	if err != nil {
		return state.Resource{}, err
	}

	r, ok := val.(state.Resource)
	if !ok {
		return state.Resource{}, fmt.Errorf("expected Resource type, got %T", val)
	}

	return r, nil
}

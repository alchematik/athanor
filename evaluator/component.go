package evaluator

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/build/component"
	"github.com/alchematik/athanor/state"
)

func (e Evaluator) Component(ctx context.Context, env state.Environment, comp component.Type) error {
	switch c := comp.(type) {
	case component.Resource:
		return e.resource(ctx, env, c)
	default:
		return fmt.Errorf("unknown component type: %T", comp)
	}
}

func (e Evaluator) resource(ctx context.Context, env state.Environment, comp component.Resource) error {
	val, err := e.Value(ctx, env, comp.Value)
	if err != nil {
		return err
	}

	r, ok := val.(state.Resource)
	if !ok {
		return fmt.Errorf("expected Resource type, got %T", val)
	}

	env.Resources[comp.Value.Identifier.Alias] = r
	return nil
}

package evaluator

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"
)

type Evaluator struct {
	ResourceAPI ResourceAPI
}

type ResourceAPI interface {
	GetResource(context.Context, state.Resource) (state.Resource, error)
}

func (e Evaluator) Evaluate(ctx context.Context, b spec.Spec) (state.Environment, error) {
	indegrees := map[string]int{}
	parentToChildren := map[string][]string{}
	for child, parents := range b.DependencyMap {
		indegrees[child] = len(parents)
		for _, parent := range parents {
			parentToChildren[parent] = append(parentToChildren[parent], child)
		}
	}

	// TODO: detect cycle.

	var queue []string
	for alias, in := range indegrees {
		if in == 0 {
			queue = append(queue, alias)
			delete(indegrees, alias)
		}
	}

	env := state.Environment{
		DependencyMap: b.DependencyMap,
		States:        map[string]state.Type{},
	}

	// TODO: parallelize.
	for len(queue) > 0 {
		var alias string
		alias, queue = queue[0], queue[1:]

		fmt.Printf("evaluating: %q\n", alias)

		comp := b.Components[alias]
		if err := e.Component(ctx, env, alias, comp); err != nil {
			return state.Environment{}, err
		}

		for _, childAlias := range parentToChildren[alias] {
			indegrees[childAlias]--
			if indegrees[childAlias] == 0 {
				queue = append(queue, childAlias)
				delete(indegrees, childAlias)
			}
		}
	}

	return env, nil
}

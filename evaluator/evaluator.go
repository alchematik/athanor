package evaluator

import (
	"context"
	"fmt"
	"github.com/alchematik/athanor/build/value"
	"github.com/alchematik/athanor/interpreter"
)

type Evaluator struct {
}

func (e Evaluator) Evaluate(ctx context.Context, env interpreter.Environment) error {
	indegrees := map[string]int{}
	for alias, val := range env.Objects {
		if r, ok := val.(value.Resource); ok {
			if _, ok := indegrees[alias]; !ok {
				indegrees[alias] = 0
			}

			for childAlias := range r.Children {
				indegrees[childAlias]++
			}
		}
	}

	fmt.Printf("indegrees>> %v\n", indegrees)

	// TODO: detect cycle.

	var queue []string
	for alias, in := range indegrees {
		if in == 0 {
			queue = append(queue, alias)
			delete(indegrees, alias)
		}
	}

	for len(queue) > 0 {
		var alias string
		alias, queue = queue[0], queue[1:]

		fmt.Printf("evaluating: %q\n", alias)

		v := env.Objects[alias]
		r := v.(value.Resource)

		var err error
		r, err = e.EvaluateResource(ctx, env, r)
		if err != nil {
			return err
		}
		env.Objects[alias] = r

		for childAlias := range r.Children {
			indegrees[childAlias]--
			if indegrees[childAlias] == 0 {
				queue = append(queue, childAlias)
				delete(indegrees, childAlias)
			}
		}
	}

	return nil
}

// TODO: This needs to either
// * Fetch the remote resource using the identifier and fill in the config and attrs fields, or
// * Fill in the config field with the static config and set attrs to something (unresolved?).
// In troduce an interface to evaulate resources?
func (e Evaluator) EvaluateResource(ctx context.Context, env interpreter.Environment, r value.Resource) (value.Resource, error) {
	id, err := e.resolveValue(ctx, env, r.Identifier)
	if err != nil {
		return value.Resource{}, err
	}

	fmt.Printf("CONFIG: %+v\n", r.Config)
	config, err := e.resolveValue(ctx, env, r.Config)
	if err != nil {
		return value.Resource{}, err
	}

	r.Identifier = id
	r.Config = config
	// r.Attrs = value.Map{
	// 	Entries: map[string]value.Type{
	// 		"bar": value.Map{
	// 			Entries: map[string]value.Type{
	// 				"foo": value.String{Value: "hi"},
	// 			},
	// 		},
	// 	},
	// }
	r.Attrs = value.Unknown{}

	return r, nil
}

func (e Evaluator) resolveValue(ctx context.Context, env interpreter.Environment, val value.Type) (value.Type, error) {
	fmt.Printf("resolving: %T : %v\n", val, val)
	switch v := val.(type) {
	case value.String:
		return v, nil
	case value.Map:
		for k, entry := range v.Entries {
			fmt.Printf("map entry: %v\n", k)
			resolved, err := e.resolveValue(ctx, env, entry)
			if err != nil {
				return nil, err
			}

			v.Entries[k] = resolved
		}
		return v, nil
	case value.Unknown:
		return value.Unknown{}, nil
	case value.Unresolved:
		if _, ok := v.Object.(value.Nil); ok {
			obj, inEnv := env.Objects[v.Name]
			if !inEnv {
				return nil, fmt.Errorf("object %q not in env", v.Name)
			}

			return obj, nil
		}

		resolved, err := e.resolveValue(ctx, env, v.Object)
		if err != nil {
			return nil, err
		}

		var m map[string]value.Type
		switch obj := resolved.(type) {
		case value.Resource:
			m = map[string]value.Type{
				"identifier": obj.Identifier,
				"config":     obj.Config,
				"attrs":      obj.Attrs,
			}
		case value.Unknown:
			return value.Unknown{}, nil
		case value.Map:
			m = obj.Entries
		default:
			return nil, fmt.Errorf("value type %T has no field %q", v.Object, v.Name)
		}

		field, ok := m[v.Name]
		if !ok {
			return nil, fmt.Errorf("property %q not set", v.Name)
		}

		return field, nil
	default:
		return nil, fmt.Errorf("unrecognized value type: %T", val)
	}
}

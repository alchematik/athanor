package selector

import (
	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"
)

type Selector struct {
	Name   string
	Parent *Selector
}

func SelectEnvironment(env state.Environment, selector Selector) (state.Environment, bool) {
	if selector.Parent == nil {
		return env, true
	}

	parent, ok := SelectEnvironment(env, *selector.Parent)
	if !ok {
		return state.Environment{}, false
	}

	st, ok := parent.States[selector.Parent.Name]
	if !ok {
		return state.Environment{}, false
	}

	envSt, ok := st.(state.Environment)
	if !ok {
		return state.Environment{}, false
	}

	return envSt, true
}

func SelectSpec(s spec.Spec, selector Selector) (spec.Spec, bool) {
	if selector.Parent == nil {
		return s, true
	}

	parent, ok := SelectSpec(s, *selector.Parent)
	if !ok {
		return spec.Spec{}, false
	}

	comp, ok := parent.Components[selector.Parent.Name]
	if !ok {
		return spec.Spec{}, false
	}

	build, ok := comp.(spec.ComponentBuild)
	if !ok {
		return spec.Spec{}, false
	}

	return build.Spec, true
}

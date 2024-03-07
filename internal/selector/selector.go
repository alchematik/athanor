package selector

import (
	"github.com/alchematik/athanor/internal/diff"
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

func SelectState(env state.Environment, sel Selector) (state.Type, bool) {
	var parent state.Environment
	if sel.Parent == nil {
		parent = env
	} else {
		s, ok := SelectState(env, *sel.Parent)
		if !ok {
			return nil, false
		}

		e, ok := s.(state.Environment)
		if !ok {
			return nil, false
		}

		parent = e
	}

	s, ok := parent.States[sel.Name]
	return s, ok
}

func SelectComponent(s spec.ComponentBuild, sel Selector) (spec.Component, bool) {
	var parent spec.ComponentBuild
	if sel.Parent == nil {
		parent = s
	} else {
		comp, ok := SelectComponent(s, *sel.Parent)
		if !ok {
			return nil, false
		}

		build, ok := comp.(spec.ComponentBuild)
		if !ok {
			return nil, false
		}

		parent = build
	}

	c, ok := parent.Spec.Components[sel.Name]
	return c, ok
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

func SelectDiffEnvironment(env diff.Environment, selector Selector) (diff.Environment, bool) {
	if selector.Parent == nil {
		return env, true
	}

	parent, ok := SelectDiffEnvironment(env, *selector.Parent)
	if !ok {
		return diff.Environment{}, false
	}

	d, ok := parent.Diffs[selector.Parent.Name]
	if !ok {
		return diff.Environment{}, false
	}

	sub, ok := d.(diff.Environment)

	return sub, true
}

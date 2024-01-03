package build

import (
	"github.com/alchematik/athanor/build/component"
	"github.com/alchematik/athanor/build/value"
)

type Build struct {
	Providers     map[string]value.Provider
	Resources     map[string]value.Resource
	DependencyMap map[string][]string
	Components    map[string]component.Type
}

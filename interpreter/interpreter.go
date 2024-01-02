package interpreter

import (
	"context"

	"github.com/alchematik/athanor/blueprint"
	"github.com/alchematik/athanor/build/value"
)

type Interpreter struct{}

type Environment struct {
	Providers     map[string]value.Provider
	Resources     map[string]value.Resource
	DependencyMap map[string][]string
}

func NewEnvironment() Environment {
	return Environment{
		Providers:     map[string]value.Provider{},
		Resources:     map[string]value.Resource{},
		DependencyMap: map[string][]string{},
	}
}

func (in Interpreter) Interpret(ctx context.Context, b blueprint.Blueprint) (value.Build, error) {
	build := value.Build{
		Providers:     map[string]value.Provider{},
		Resources:     map[string]value.Resource{},
		DependencyMap: map[string][]string{},
	}
	for _, st := range b.Stmts {
		if err := in.Stmt(ctx, build, st); err != nil {
			return value.Build{}, err
		}
	}

	return build, nil
}

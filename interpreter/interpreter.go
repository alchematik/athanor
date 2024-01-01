package interpreter

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/blueprint"
	"github.com/alchematik/athanor/blueprint/stmt"
	"github.com/alchematik/athanor/build/value"
)

type Interpreter struct{}

type Environment struct {
	Providers     map[string]value.Provider
	Resources     map[string]value.Resource
	DependencyMap map[string][]string
}

func (in Interpreter) Interpret(ctx context.Context, env Environment, b blueprint.Blueprint) error {
	for _, st := range b.Stmts {
		switch s := st.(type) {
		case stmt.Provider:
			if err := in.ProviderStmt(ctx, env, s); err != nil {
				return err
			}
		case stmt.Resource:
			if err := in.ResourceStmt(ctx, env, s); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown stmt %T", st)
		}
	}

	return nil
}

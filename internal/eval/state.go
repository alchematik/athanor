package eval

import (
	"context"
	"encoding/gob"
	"fmt"
	"log/slog"

	"github.com/alchematik/athanor/internal/dag"
	"github.com/alchematik/athanor/internal/state"
)

func init() {
	gob.Register(map[string]any{})
}

type StateEvaluator struct {
	Iter            *dag.Iterator
	Logger          *slog.Logger
	providerManager ProviderManager
}

type ProviderManager interface {
	ProviderPlugin(state.Provider) (ProviderPlugin, error)
}

type ProviderPlugin interface {
	Get(context.Context, any) (state.Resource, error)
}

func (e *StateEvaluator) Next() []string {
	return e.Iter.Next()
}

func (e *StateEvaluator) Eval(ctx context.Context, s *state.State, stmt any) error {
	switch stmt := stmt.(type) {
	case state.StmtResource:
		current, ok := s.Resource(stmt.ID)
		if !ok {
			return fmt.Errorf("resource not in state: %s", stmt.ID)
		}

		if e.Iter.Visited(stmt.ID) {
			return e.Iter.Done(stmt.ID)
		}

		current.ToEvaluating()

		if err := e.Iter.Start(stmt.ID); err != nil {
			return err
		}

		r, err := stmt.Resource.Eval(ctx, s)
		if err != nil {
			current.ToError(err)
			return nil
		}

		// TODO: Handle cases where not exist
		current.ToDone(r, true)
		return nil
	case state.StmtBuild:
		current, ok := s.Build(stmt.ID)
		if !ok {
			return fmt.Errorf("build not in state: %s", stmt.ID)
		}

		// TODO: this assumes that if we've visited already, we're done.
		// this might not be true, especially with watchers.
		if e.Iter.Visited(stmt.ID) {
			current.ToDone()
			return e.Iter.Done(stmt.ID)
		}

		// TODO: handle case where doesn't exist.
		current.SetExists(true)

		current.ToEvaluating()
		return e.Iter.Start(stmt.ID)
	default:
		return fmt.Errorf("unsupported component type: %T", stmt)
	}
}

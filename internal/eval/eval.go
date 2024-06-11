package eval

import (
	"fmt"

	"log/slog"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/dag"
	"github.com/alchematik/athanor/internal/state"
)

func NewTargetEvaluator(iter *dag.Iterator) *TargetEvaluator {
	return &TargetEvaluator{iter: iter}
}

type TargetEvaluator struct {
	iter   *dag.Iterator
	Logger *slog.Logger
}

func (e *TargetEvaluator) Next() []string {
	return e.iter.Next()
}

func (e *TargetEvaluator) Eval(g *state.Global, stmt any) error {
	switch stmt := stmt.(type) {
	case ast.StmtResource:
		s := g.Target()
		current, ok := s.ResourceState(stmt.ID)
		if !ok {
			return fmt.Errorf("resource not in state: %s", stmt.ID)
		}

		if e.iter.Visited(stmt.ID) {
			return e.iter.Done(stmt.ID)
		}

		current.ToEvaluating()

		if err := e.iter.Start(stmt.ID); err != nil {
			return err
		}

		r, err := stmt.Resource.Eval(s)
		if err != nil {
			current.ToError(err)
			// TODO: is there a way to short-curcuit?
			return nil
		}

		current.ToDone(r)
		var action state.ComponentAction
		if r.Exists {
			action = state.ComponentActionCreate
		} else {
			action = state.ComponentActionDelete
		}
		current.SetComponentAction(action)

		return nil
	case ast.StmtBuild:
		s := g.Target()
		current, ok := s.BuildState(stmt.ID)
		if !ok {
			return fmt.Errorf("build not in state: %s", stmt.ID)
		}

		// TODO: this assumes that if we've visited already, we're done.
		// this might not be true, especially with watchers.
		if e.iter.Visited(stmt.ID) {
			current.ToDone()
			return e.iter.Done(stmt.ID)
		}

		current.ToEvaluating()
		return e.iter.Start(stmt.ID)
	default:
		return fmt.Errorf("unsupported component type: %T", stmt)
	}
}

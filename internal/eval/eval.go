package eval

import (
	"context"
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
	api    *TargetAPI
}

func (e *TargetEvaluator) Next() []string {
	return e.iter.Next()
}

func (e *TargetEvaluator) Eval(ctx context.Context, g *state.Global, stmt any) error {
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

		r, err := stmt.Resource.Eval(ctx, e.api, s)
		if err != nil {
			current.ToError(err)
			// TODO: is there a way to short-curcuit?
			return nil
		}

		exists, err := stmt.Exists.Eval(ctx, e.api, s)
		if err != nil {
			current.ToError(err)
			return nil
		}

		parent, ok := s.BuildState(stmt.BuildID)
		if ok && !parent.GetExists() {
			exists = false
		}

		current.ToDone(r, exists)

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

		exists, err := stmt.Exists.Eval(ctx, e.api, s)
		if err != nil {
			current.ToError(err)
			return nil
		}

		parent, ok := s.BuildState(stmt.BuildID)
		if ok && !parent.GetExists() {
			exists = false
		}

		current.SetExists(exists)

		current.ToEvaluating()
		return e.iter.Start(stmt.ID)
	default:
		return fmt.Errorf("unsupported component type: %T", stmt)
	}
}

type TargetAPI struct {
}

func (a *TargetAPI) EvalResource(ctx context.Context, res *state.Resource) error {
	return nil
}

package eval

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alchematik/athanor/internal/dag"
	"github.com/alchematik/athanor/internal/plan"
)

func NewPlanEvaluator(iter *dag.Iterator, logger *slog.Logger) *PlanEvaluator {
	return &PlanEvaluator{iter: iter, logger: logger}
}

type PlanEvaluator struct {
	iter   *dag.Iterator
	logger *slog.Logger
}

func (e *PlanEvaluator) Next() []string {
	return e.iter.Next()
}

func (e *PlanEvaluator) Eval(ctx context.Context, p *plan.Plan, stmt any) error {
	switch stmt := stmt.(type) {
	case plan.StmtResource:
		current, ok := p.Resource(stmt.ID)
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

		exists, err := stmt.Exists.Eval(ctx, p)
		if err != nil {
			current.ToError(err)
			return nil
		}

		r, err := stmt.Resource.Eval(ctx, p)
		if err != nil {
			// TODO: handle not found.
			current.ToError(err)
			// TODO: is there a way to short-curcuit?
			return nil
		}

		parent, ok := p.Build(stmt.BuildID)
		if ok {
			// Parent exists value is known, and it's set to false. Child resource exists should be false also.
			parentExists := parent.GetExists()
			if !exists.Unknown && !parentExists.Unknown && !parentExists.Value {
				exists.Value = false
			}
		}

		current.ToDone(r, exists)

		return nil
	case plan.StmtBuild:
		current, ok := p.Build(stmt.ID)
		if !ok {
			return fmt.Errorf("build not in state: %s", stmt.ID)
		}

		// TODO: this assumes that if we've visited already, we're done.
		// this might not be true, especially with watchers.
		if e.iter.Visited(stmt.ID) {
			current.ToDone()
			return e.iter.Done(stmt.ID)
		}

		exists, err := stmt.Exists.Eval(ctx, p)
		if err != nil {
			current.ToError(err)
			return nil
		}

		parent, ok := p.Build(stmt.BuildID)
		if ok {
			// Parent exists value is known, and it's set to false. Child resource exists should be false also.
			parentExists := parent.GetExists()
			if !exists.Unknown && !parentExists.Unknown && !parentExists.Value {
				exists.Value = false
			}
		}

		current.SetExists(exists)

		current.ToEvaluating()
		return e.iter.Start(stmt.ID)
	default:
		return fmt.Errorf("unsupported component type: %T", stmt)
	}
}

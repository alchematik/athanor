package eval

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alchematik/athanor/internal/dag"
	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/plan"
	"github.com/alchematik/athanor/internal/state"
)

type DiffEvaluator struct {
	Iter   *dag.Iterator
	Logger *slog.Logger
}

func (e *DiffEvaluator) Next() []string {
	return e.Iter.Next()
}

func (e *DiffEvaluator) Eval(ctx context.Context, d *diff.DiffResult, stmt any) error {
	switch stmt := stmt.(type) {
	case diff.StmtResource:
		current, ok := d.Resource(stmt.ID)
		if !ok {
			return fmt.Errorf("resource not in diff: %s", stmt.ID)
		}

		if e.Iter.Visited(stmt.ID) {
			return e.Iter.Done(stmt.ID)
		}

		current.ToEvaluating()

		if err := e.Iter.Start(stmt.ID); err != nil {
			return err
		}

		r, err := stmt.Resource.Eval(ctx, d)
		if err != nil {
			current.ToError(err)
			return nil
		}

		// TODO: This doesn't make sense because the state exists will always be false.
		// Need to get rid of the resource expr and move the identifier, config, etc into the statement.
		exists, err := stmt.Exists.Eval(ctx, d)
		if err != nil {
			current.ToError(err)
			return nil
		}

		current.ToDone(r, exists)
		return nil
	case diff.StmtBuild:
		current, ok := d.Build(stmt.ID)
		if !ok {
			return fmt.Errorf("build not in state: %s", stmt.ID)
		}

		if e.Iter.Visited(stmt.ID) {
			s, _ := d.State.Build(stmt.ID)
			p, _ := d.Plan.Build(stmt.ID)
			stateExists := s.GetExists()
			planExists := p.GetExists()
			expr := diff.ExprLiteral[bool]{
				Plan:  plan.ExprLiteral[bool]{Value: planExists.Value},
				State: state.ExprLiteral[bool]{Value: stateExists},
			}

			// TODO: Use these values
			exists, err := expr.Eval(ctx, d)
			e.Logger.Info("got build exists diff", "exists", exists, "err", err)

			current.ToDone()
			return e.Iter.Done(stmt.ID)
		}

		current.ToEvaluating()
		return e.Iter.Start(stmt.ID)
	default:
		return fmt.Errorf("unsupported component type: %T", stmt)
	}
}

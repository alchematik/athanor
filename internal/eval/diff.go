package eval

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alchematik/athanor/internal/dag"
	"github.com/alchematik/athanor/internal/diff"
)

type DiffEvaluator struct {
	Iter   *dag.Iterator
	Logger *slog.Logger
}

func (e *DiffEvaluator) Next() []string {
	return e.Iter.Next()
}

func (e *DiffEvaluator) Eval(ctx context.Context, d *diff.Diff, stmt any) error {
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

		exists, err := stmt.Exists.Eval(ctx, d)
		if err != nil {
			current.ToError(err)
			return nil
		}

		e.Logger.Info("resource >>>>>>>>>>>>>>", "resource", r)

		current.ToDone(r, exists)
		return nil
	case diff.StmtBuild:
		current, ok := d.Build(stmt.ID)
		if !ok {
			return fmt.Errorf("build not in state: %s", stmt.ID)
		}

		if e.Iter.Visited(stmt.ID) {
			current.ToDone()
			return e.Iter.Done(stmt.ID)
		}

		current.ToEvaluating()
		return e.Iter.Start(stmt.ID)
	default:
		return fmt.Errorf("unsupported component type: %T", stmt)
	}
}

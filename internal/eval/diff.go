package eval

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/alchematik/athanor/internal/dag"
	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/plan"
	"github.com/alchematik/athanor/provider"

	"github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
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

		planCurrent, ok := d.Plan.Resource(stmt.ID)
		if !ok {
			return fmt.Errorf("resource not in plan: %s", stmt.ID)
		}

		planExists, err := stmt.PlanExists.Eval(ctx, d.Plan)
		if err != nil {
			current.ToError(err)
			return nil
		}
		planCurrent.SetExists(planExists)

		planType, err := stmt.PlanType.Eval(ctx, d.Plan)
		if err != nil {
			current.ToError(err)
			return nil
		}
		planCurrent.SetType(planType)

		planProvider, err := stmt.PlanProvider.Eval(ctx, d.Plan)
		if err != nil {
			current.ToError(err)
			return nil
		}
		planCurrent.SetProvider(planProvider)

		planIdentifier, err := stmt.PlanIdentifier.Eval(ctx, d.Plan)
		if err != nil {
			current.ToError(err)
			return nil
		}
		planCurrent.SetIdentifier(planIdentifier)

		planConfig, err := stmt.PlanConfig.Eval(ctx, d.Plan)
		if err != nil {
			current.ToError(err)
			return nil
		}
		planCurrent.SetConfig(planConfig)

		stateCurrent, ok := d.State.Resource(stmt.ID)
		if !ok {
			return fmt.Errorf("resource not in state: %s", stmt.ID)
		}

		t, err := stmt.Type.Eval(ctx, d.State)
		if err != nil {
			current.ToError(err)
			return nil
		}
		stateCurrent.SetType(t)

		id, err := stmt.Identifier.Eval(ctx, d.State)
		if err != nil {
			current.ToError(err)
			return nil
		}
		stateCurrent.SetIdentifier(id)
		current.SetIdentifier(id)

		prov, err := stmt.Provider.Eval(ctx, d.State)
		if err != nil {
			current.ToError(err)
			return nil
		}
		stateCurrent.SetProvider(prov)
		current.SetProvider(prov)

		// TODO: Extract and use provider to determine plugin.
		client := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig: plugin.HandshakeConfig{
				ProtocolVersion:  1,
				MagicCookieKey:   "BASIC_PLUGIN",
				MagicCookieValue: "hello",
			},
			Plugins: map[string]plugin.Plugin{
				"provider": &provider.Plugin{},
			},
			Cmd:    exec.Command(".provider/google-cloud-v0.0.1"),
			Logger: hclog.NewNullLogger(),
		})
		defer client.Kill()

		c, err := client.Client()
		if err != nil {
			current.ToError(err)
			return nil
		}

		pr, err := c.Dispense("provider")
		if err != nil {
			current.ToError(err)
			return nil
		}

		providerClient, ok := pr.(*provider.Client)
		if !ok {
			current.ToError(fmt.Errorf("invalid provider client: %T", pr))
			return nil
		}

		res, err := providerClient.Get(ctx, provider.GetResourceRequest{
			Type:       t,
			Identifier: id,
		})
		if err != nil {
			current.ToError(err)
			return nil
		}

		// TODO: Handle case where doesn't exist.
		stateCurrent.SetExists(true)
		stateCurrent.SetConfig(res.Resource.Config)
		stateCurrent.SetAttributes(res.Resource.Attrs)

		// TODO: diff and set on current.
		existsDiff, err := diff.DiffLiteral[bool](
			diff.Emptyable[plan.Maybe[bool]]{Value: planExists},
			diff.Emptyable[bool]{Value: true},
		)
		current.SetExists(existsDiff)

		configDiff, err := diff.DiffAny(
			diff.Emptyable[plan.Maybe[any]]{Value: planConfig},
			diff.Emptyable[any]{Value: res.Resource.Config},
		)
		current.SetConfig(configDiff)

		action := existsDiff.Action
		if action == diff.ActionNoop {
			action = configDiff.Action
		}

		current.SetAction(action)

		current.ToDone()
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
			exists, err := diff.DiffLiteral[bool](
				diff.Emptyable[plan.Maybe[bool]]{Value: planExists},
				diff.Emptyable[bool]{Value: stateExists},
			)
			if err != nil {
				return err
			}

			// TODO: Use these values
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

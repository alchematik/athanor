package eval

import (
	"context"
	"encoding/gob"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/alchematik/athanor/internal/dag"
	"github.com/alchematik/athanor/internal/state"
	"github.com/alchematik/athanor/provider"

	"github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
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

		t, err := stmt.Type.Eval(ctx, s)
		if err != nil {
			current.ToError(err)
			return nil
		}
		current.SetType(t)

		// TODO: Use provider to initialize plugin client.
		prov, err := stmt.Provider.Eval(ctx, s)
		if err != nil {
			current.ToError(err)
			return nil
		}
		current.SetProvider(prov)

		id, err := stmt.Identifier.Eval(ctx, s)
		if err != nil {
			current.ToError(err)
			return nil
		}
		current.SetIdentifier(id)

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

		// TODO: Handle case where doesn't exist
		current.SetExists(true)
		current.SetConfig(res.Resource.Config)
		current.SetAttributes(res.Resource.Attrs)

		current.ToDone()
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

		// TODO: handle case where doesn't exist by checking all resources and sub builds in build.
		current.SetExists(true)

		current.ToEvaluating()
		return e.Iter.Start(stmt.ID)
	default:
		return fmt.Errorf("unsupported component type: %T", stmt)
	}
}

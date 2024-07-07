package state

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/alchematik/athanor/provider"

	"github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
)

type StmtBuild struct {
	ID           string
	Name         string
	BuildID      string
	Exists       Expr[bool]
	RuntimeInput Expr[map[string]any]
	Stmts        []any
}

type StmtResource struct {
	ID       string
	Name     string
	BuildID  string
	Exists   Expr[bool]
	Resource Expr[Resource]
}

type Expr[T any] interface {
	Eval(context.Context, *State) (T, error)
}

type Provider struct {
	Name    string
	Version string
}

type Resource struct {
	Type       string
	Provider   Provider
	Identifier any
	Config     any
	Attrs      any
}

type ExprAny[T any] struct {
	Value Expr[T]
}

func (e ExprAny[T]) Eval(ctx context.Context, s *State) (any, error) {
	out, err := e.Value.Eval(ctx, s)
	if err != nil {
		return nil, err
	}

	return out, nil
}

type ExprLiteral[T any] struct {
	Value T
}

func (e ExprLiteral[T]) Eval(_ context.Context, _ *State) (T, error) {
	return e.Value, nil
}

type ExprMap map[Expr[string]]Expr[any]

func (e ExprMap) Eval(ctx context.Context, s *State) (map[string]any, error) {
	m := map[string]any{}
	for k, v := range e {
		key, err := k.Eval(ctx, s)
		if err != nil {
			return nil, err
		}

		val, err := v.Eval(ctx, s)
		if err != nil {
			return nil, err
		}

		m[key] = val
	}

	return m, nil
}

type ExprResource struct {
	Name       string
	Type       Expr[string]
	Provider   Expr[Provider]
	Identifier Expr[any]
	Config     Expr[any]
}

func (e ExprResource) Eval(ctx context.Context, s *State) (Resource, error) {
	id, err := e.Identifier.Eval(ctx, s)
	if err != nil {
		return Resource{}, err
	}

	t, err := e.Type.Eval(ctx, s)
	if err != nil {
		return Resource{}, err
	}

	p, err := e.Provider.Eval(ctx, s)
	if err != nil {
		return Resource{}, err
	}

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
		return Resource{}, err
	}

	pr, err := c.Dispense("provider")
	if err != nil {
		return Resource{}, err
	}

	providerClient, ok := pr.(*provider.Client)
	if !ok {
		return Resource{}, fmt.Errorf("invalid provider client: %T", p)
	}

	res, err := providerClient.Get(ctx, provider.GetResourceRequest{
		Type:       t,
		Identifier: id,
	})

	return Resource{
		Type:       t,
		Provider:   p,
		Identifier: id,
		Config:     res.Resource.Config,
		Attrs:      res.Resource.Attrs,
	}, nil
}

type ExprProvider struct {
	Name    Expr[string]
	Version Expr[string]
}

func (e ExprProvider) Eval(ctx context.Context, s *State) (Provider, error) {
	name, err := e.Name.Eval(ctx, s)
	if err != nil {
		return Provider{}, err
	}

	version, err := e.Version.Eval(ctx, s)
	if err != nil {
		return Provider{}, err
	}

	return Provider{
		Name:    name,
		Version: version,
	}, nil
}

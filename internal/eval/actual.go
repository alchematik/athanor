package eval

import (
	"context"
	"encoding/gob"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/alchematik/athanor/internal/state"
	"github.com/alchematik/athanor/provider"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

func init() {
	gob.Register(map[string]any{})
}

type ActualAPI struct {
	logger *slog.Logger
}

func (a *ActualAPI) EvalResource(ctx context.Context, res *state.Resource) error {
	p, err := unwrapProvider(res.Provider)
	if err != nil {
		return err
	}

	t, ok := res.Type.Unwrap()
	if !ok {
		return fmt.Errorf("resource type is required")
	}

	id, err := unwrapAny(res.Identifier)
	if err != nil {
		return fmt.Errorf("identifier is required: %s", err)
	}

	var handshakeConfig = plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "BASIC_PLUGIN",
		MagicCookieValue: "hello",
	}

	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		"provider": &provider.Plugin{},
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
		Cmd:             exec.Command(fmt.Sprintf("./.provider/%s-%s", p.Name, p.Version)),
		Logger:          hclog.NewNullLogger(),
	})

	rpcClient, err := client.Client()
	if err != nil {
		return err
	}

	raw, err := rpcClient.Dispense("provider")
	if err != nil {
		return err
	}

	prov, ok := raw.(*provider.Client)
	if !ok {
		return fmt.Errorf("invalid provider: %T", raw)
	}

	resp, err := prov.Get(ctx, provider.GetResourceRequest{
		Type:       t,
		Identifier: id,
	})
	if err != nil {
		return err
	}

	// TODO: Handle nil
	res.Attributes = wrapAny(resp.Resource.Attrs)
	res.Config = wrapAny(resp.Resource.Config)
	// res.Attributes = state.Maybe[any]{
	// 	Value: resp.Resource.Attrs,
	// }
	// res.Config = state.Maybe[any]{
	// 	Value: resp.Resource.Config,
	// }
	client.Kill()

	return nil
}

type Provider struct {
	Name    string
	Version string
}

func unwrapProvider(in state.Maybe[state.Provider]) (Provider, error) {
	p, ok := in.Unwrap()
	if !ok {
		return Provider{}, fmt.Errorf("provider is required")
	}

	name, ok := p.Name.Unwrap()
	if !ok {
		return Provider{}, fmt.Errorf("provider name is required")
	}

	version, ok := p.Version.Unwrap()
	if !ok {
		return Provider{}, fmt.Errorf("provider version is required")
	}

	return Provider{
		Name:    name,
		Version: version,
	}, nil
}

func unwrapAny(in state.Maybe[any]) (any, error) {
	a, ok := in.Unwrap()
	if !ok {
		return nil, fmt.Errorf("value must be known")
	}

	switch val := a.(type) {
	case state.Maybe[any]:
		return unwrapAny(val)
	case state.Maybe[string]:
		return unwrapLiteral[string](val)
	case state.Maybe[bool]:
		return unwrapLiteral[bool](val)
	// case state.Maybe[state.Resource]:
	case state.Maybe[state.Provider]:
		return unwrapProvider(val)
	case state.Maybe[map[state.Maybe[string]]state.Maybe[any]]:
		return unwrapMap(val)
	default:
		return nil, fmt.Errorf("unwrap: unknown type: %T", a)
	}
}

func unwrapLiteral[T any](in state.Maybe[T]) (T, error) {
	var val T
	out, ok := in.Unwrap()
	if !ok {
		return val, fmt.Errorf("value must be known")
	}

	return out, nil
}

type Resource struct {
	Provider
}

func unwrapMap(in state.Maybe[map[state.Maybe[string]]state.Maybe[any]]) (map[string]any, error) {
	m, ok := in.Unwrap()
	if !ok {
		return nil, fmt.Errorf("map value must be known")
	}

	out := map[string]any{}
	for k, v := range m {
		str, err := unwrapLiteral[string](k)
		if err != nil {
			return nil, err
		}

		val, err := unwrapAny(v)
		if err != nil {
			return nil, err
		}

		out[str] = val
	}

	return out, nil
}

func wrapMap(in map[string]any) state.Maybe[map[state.Maybe[string]]state.Maybe[any]] {
	out := map[state.Maybe[string]]state.Maybe[any]{}
	for k, v := range in {
		out[state.Maybe[string]{Value: k}] = wrapAny(v)
	}

	return state.Maybe[map[state.Maybe[string]]state.Maybe[any]]{Value: out}
}

func wrapAny(in any) state.Maybe[any] {
	switch val := in.(type) {
	case string:
		return state.Maybe[any]{Value: wrapLiteral[string](val)}
	case bool:
		return state.Maybe[any]{Value: wrapLiteral[bool](val)}
	case map[string]any:
		return state.Maybe[any]{Value: wrapMap(val)}
	default:
		panic(fmt.Sprintf("unknown type to wrap: %T", val))
	}
}

func wrapLiteral[T any](in T) state.Maybe[T] {
	return state.Maybe[T]{Value: in}
}

package plugin

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"

	backendpb "github.com/alchematik/athanor/internal/gen/go/proto/provider/v1"
	"github.com/alchematik/athanor/internal/state"

	"github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type Provider struct {
	Logger hclog.Logger

	clients map[string]backendpb.ProviderClient
	lock    *sync.Mutex
}

func NewProvider(logger hclog.Logger) *Provider {
	return &Provider{
		Logger:  logger,
		clients: map[string]backendpb.ProviderClient{},
		lock:    &sync.Mutex{},
	}
}

func (p *Provider) Client(provider state.Provider) (backendpb.ProviderClient, error) {
	var dir string
	switch r := provider.Repo.(type) {
	case state.RepoLocal:
		dir = r.Path
	default:
		return nil, fmt.Errorf("invalid repo type: %T", provider.Repo)
	}

	pluginPath := filepath.Join(dir, provider.Name, provider.Version, "provider")

	p.lock.Lock()
	c, ok := p.clients[pluginPath]
	p.lock.Unlock()
	if ok {
		return c, nil
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "COOKIE",
			MagicCookieValue: "hi",
		},
		Plugins: map[string]plugin.Plugin{
			"provider": &ProviderPlugin{},
		},
		Cmd:              exec.Command("sh", "-c", pluginPath),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           p.Logger,
	})

	dispensor, err := client.Client()
	if err != nil {
		return nil, err
	}

	rawPlug, err := dispensor.Dispense("provider")
	if err != nil {
		return nil, err
	}

	plug, ok := rawPlug.(backendpb.ProviderClient)
	if !ok {
		return nil, fmt.Errorf("expected BackendClient, got %T", rawPlug)
	}

	p.lock.Lock()
	p.clients[pluginPath] = plug
	p.lock.Unlock()

	return plug, nil
}

type ProviderPlugin struct {
	plugin.Plugin

	BackendServer backendpb.ProviderServer
}

func (p *ProviderPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	backendpb.RegisterProviderServer(s, p.BackendServer)
	return nil
}

func (p *ProviderPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return backendpb.NewProviderClient(conn), nil
}

package plugin

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sync"

	"github.com/alchematik/athanor/internal/dependency"
	backendpb "github.com/alchematik/athanor/internal/gen/go/proto/provider/v1"
	"github.com/alchematik/athanor/internal/repo"
	"github.com/alchematik/athanor/internal/state"

	"github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type Provider struct {
	Logger hclog.Logger

	clients           map[string]backendpb.ProviderClient
	lock              *sync.Mutex
	dependencyManager *dependency.Manager
}

func NewProvider(logger hclog.Logger, depManager *dependency.Manager) *Provider {
	return &Provider{
		Logger:            logger,
		clients:           map[string]backendpb.ProviderClient{},
		lock:              &sync.Mutex{},
		dependencyManager: depManager,
	}
}

func (p *Provider) Client(ctx context.Context, provider state.Provider) (backendpb.ProviderClient, error) {
	var src any
	switch s := provider.Repo.(type) {
	case repo.PluginSourceLocal:
		src = dependency.SourceLocal{Path: s.Path}
	case repo.PluginSourceGitHubRelease:
		src = dependency.SourceGitHubRelease{
			RepoOwner: s.RepoOwner,
			RepoName:  s.RepoName,
			Name:      s.Name,
		}
	default:
		return nil, fmt.Errorf("invalid source type: %T", s)
	}

	dep := dependency.BinDependency{
		Type:   "provider",
		Source: src,
		OS:     runtime.GOOS,
		Arch:   runtime.GOARCH,
	}

	binPath, err := p.dependencyManager.FetchBinDependency(ctx, dep)
	if err != nil {
		return nil, err
	}

	p.lock.Lock()
	c, ok := p.clients[binPath]
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
		Cmd:              exec.Command("sh", "-c", binPath),
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
	p.clients[binPath] = plug
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

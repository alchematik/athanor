package plugin

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/alchematik/athanor/backend"
	backendpb "github.com/alchematik/athanor/internal/gen/go/proto/provider/v1"
	"github.com/alchematik/athanor/state"

	plugin "github.com/hashicorp/go-plugin"
)

type Provider struct {
	Dir string
}

func (p Provider) Client(provider state.Provider) (backendpb.ProviderClient, error) {
	pluginPath := filepath.Join(p.Dir, provider.Name, provider.Version, "provider")

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: backend.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"backend": &backend.Plugin{},
		},
		Cmd:              exec.Command("sh", "-c", pluginPath),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
	})

	dispensor, err := client.Client()
	if err != nil {
		return nil, err
	}

	rawPlug, err := dispensor.Dispense("backend")
	if err != nil {
		return nil, err
	}

	plug, ok := rawPlug.(backendpb.ProviderClient)
	if !ok {
		return nil, fmt.Errorf("expected BackendClient, got %T", rawPlug)
	}

	return plug, nil
}

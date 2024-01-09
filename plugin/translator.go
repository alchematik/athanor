package plugin

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	translatorpb "github.com/alchematik/athanor/internal/gen/go/proto/translator/v1"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type Translator struct {
	Dir string
}

func (t Translator) Client(name, version string) (translatorpb.TranslatorClient, error) {
	pluginPath := filepath.Join(t.Dir, name, version, "translator")

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "COOKIE",
			MagicCookieValue: "hi",
		},
		Plugins: map[string]plugin.Plugin{
			"translator": &TranslatorPlugin{},
		},
		Cmd:              exec.Command("sh", "-c", pluginPath),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           hclog.NewNullLogger(),
	})

	dispensor, err := client.Client()
	if err != nil {
		return nil, err
	}

	rawPlug, err := dispensor.Dispense("translator")
	if err != nil {
		return nil, err
	}

	plug, ok := rawPlug.(translatorpb.TranslatorClient)
	if !ok {
		return nil, fmt.Errorf("expected TranslatorClient, got %T", rawPlug)
	}

	return plug, nil
}

type TranslatorPlugin struct {
	plugin.Plugin

	TranslatorServer translatorpb.TranslatorServer
}

func (p *TranslatorPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	translatorpb.RegisterTranslatorServer(s, p.TranslatorServer)
	return nil
}

func (p *TranslatorPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return translatorpb.NewTranslatorClient(conn), nil
}

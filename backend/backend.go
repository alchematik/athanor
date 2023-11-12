package backend

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	backendpb "github.com/alchematik/athanor/internal/gen/go/proto/backend/v1"
)

var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN_BACKEND",
	MagicCookieValue: "hello_backend",
}

type Plugin struct {
	plugin.Plugin

	BackendServer backendpb.BackendServer
}

func (p *Plugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	backendpb.RegisterBackendServer(s, p.BackendServer)
	return nil
}

func (p *Plugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return backendpb.NewBackendClient(conn), nil
}

package provider

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

type GetResourceRequest struct {
	Type       string
	Identifier any
}

type GetResourceResponse struct {
	Resource Resource
}

type Resource struct {
	Type       string
	Identifier any
	Config     any
	Attrs      any
}

type Provider interface {
	Get(GetResourceRequest) (GetResourceResponse, error)
}

type Plugin struct {
	Impl Provider
}

func (p *Plugin) Server(*plugin.MuxBroker) (any, error) {
	return &Server{Impl: p.Impl}, nil
}

func (*Plugin) Client(b *plugin.MuxBroker, c *rpc.Client) (any, error) {
	return &Client{client: c}, nil
}

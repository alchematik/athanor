package provider

import (
	"context"
	"net/rpc"
)

type Client struct {
	client *rpc.Client
}

func (c *Client) Get(_ context.Context, req GetResourceRequest) (GetResourceResponse, error) {
	var res GetResourceResponse
	if err := c.client.Call("Plugin.Get", req, &res); err != nil {
		return GetResourceResponse{}, err
	}

	return res, nil
}

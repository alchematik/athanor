package resource_policy

import (
	"context"

	"errors"

	"github.com/alchematik/athanor/provider"
)

type Client interface {
	GetResourcePolicy(context.Context, *Identifier) (*ResourcePolicy, error)

	CreateResourcePolicy(context.Context, *Identifier, *Config) error
}

type ResourcePolicy struct {
	Identigier *Identifier
	Config     *Config
}

func GetResource(ctx context.Context, client Client, id *Identifier) (*provider.Resource, error) {
	r := &provider.Resource{Identifier: id}
	out, err := client.GetResourcePolicy(ctx, id)
	if err != nil {
		if errors.Is(err, provider.NotFoundError) {
			r.State = provider.ResourceStateNotExists
			return r, nil
		}

		return nil, err
	}

	r.Config = out.Config
	return r, nil
}

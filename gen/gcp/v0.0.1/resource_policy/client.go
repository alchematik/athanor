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
	Identifier *Identifier
	Config     *Config
}

func GetResource(ctx context.Context, client Client, identifier []provider.FieldValue) (*provider.Resource, error) {
	r := &provider.Resource{}
	id := FieldValuesToIdentifier(identifier)
	_, err := client.GetResourcePolicy(ctx, id)
	if err != nil {
		if errors.Is(err, provider.NotFoundError) {
			r.State = provider.ResourceStateNotExists
			return r, nil
		}

		return nil, err
	}

	// r.Config = out.Config
	return r, nil
}

package bucket_object

import (
	"context"

	"errors"

	"github.com/alchematik/athanor/provider"
)

type Client interface {
	GetBucketObject(context.Context, *Identifier) (*BucketObject, error)

	CreateBucketObject(context.Context, *Identifier, *Config) error
}

type BucketObject struct {
	Identigier *Identifier
	Config     *Config
}

func GetResource(ctx context.Context, client Client, id *Identifier) (*provider.Resource, error) {
	r := &provider.Resource{Identifier: id}
	out, err := client.GetBucketObject(ctx, id)
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

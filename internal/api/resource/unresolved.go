package resource

import (
	"context"

	"github.com/alchematik/athanor/state"
)

type Unresolved struct {
}

func (u *Unresolved) GetResource(ctx context.Context, r state.Resource) (state.Resource, error) {
	return state.Resource{
		Provider:   r.Provider,
		Identifier: r.Identifier,
		Config:     r.Config,
		Exists:     r.Exists,
		Attrs: state.Unknown{
			Name: "attrs",
			Object: state.ResourceRef{
				Alias: r.Identifier.Alias,
			},
		},
	}, nil
}

package resource_policy

import (
	"github.com/alchematik/athanor/provider"
)

type Op struct {
	Type       string
	Identifier *Identifier
	Version    string
	Config     Config
}

func (o *Op) ForIdentifier() provider.Identifier {
	return o.Identifier
}

func (o *Op) ForVersion() string {
	return o.Version
}

func (o *Op) Apply(r *provider.Resource) {
	r.State = provider.ResourceStateExists
	r.Identifier = o.Identifier
	r.Config = o.Config
}

type Config struct {
}

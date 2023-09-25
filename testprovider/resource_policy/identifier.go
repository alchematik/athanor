package resource_policy

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/alchematik/athanor/identifier"

	"github.com/alchematik/athanor/testprovider/bucket"
)

// Identifier is the identifier for a resource_policy.
type Identifier struct {
	// Name is the name of the resource policy.
	Name string

	// Resource is the resource that the policy belongs to.
	Resource any
}

type HCLIdentifier struct {
	Name string `hcl:"name" cty:"name"`

	Resource identifier.HCLIdentifier `hcl:"resource" cty:"resource"`
}

func (id *HCLIdentifier) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{
		"name":     cty.String,
		"resource": id.Resource.CtyType(),
	})
}

func (id *HCLIdentifier) ToCtyValue() (cty.Value, error) {
	return gocty.ToCtyValue(id, id.CtyType())
}

func (id *HCLIdentifier) ToIdentifier() *Identifier {
	out := &Identifier{}

	out.Name = id.Name

	switch t := id.Resource.(type) {
	case *bucket.HCLIdentifier:
		out.Resource = t.ToIdentifier()
	}

	return out
}

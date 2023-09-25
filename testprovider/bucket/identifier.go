package bucket

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/zclconf/go-cty/cty/gocty"
)

// Identifier is the identifier for a bucket.
type Identifier struct {
	// Account is the account that the bucket belongs to.
	Account string

	// Region is the region that the bucket belongs in.
	Region string

	// Name is the name of the bucket.
	Name string
}

type HCLIdentifier struct {
	Account string `hcl:"account" cty:"account"`

	Region string `hcl:"region" cty:"region"`

	Name string `hcl:"name" cty:"name"`
}

func (id *HCLIdentifier) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{
		"account": cty.String,
		"region":  cty.String,
		"name":    cty.String,
	})
}

func (id *HCLIdentifier) ToCtyValue() (cty.Value, error) {
	return gocty.ToCtyValue(id, id.CtyType())
}

func (id *HCLIdentifier) ToIdentifier() *Identifier {
	out := &Identifier{}

	out.Account = id.Account

	out.Region = id.Region

	out.Name = id.Name

	return out
}

package bucket_object

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/alchematik/athanor/testprovider/bucket"
)

// Identifier is the identifier for a bucket_object.
type Identifier struct {
	// Bucket is the bucket that the object belongs to.
	Bucket *bucket.Identifier

	// Name is the name of the bucket_object.
	Name string
}

type HCLIdentifier struct {
	Bucket *bucket.HCLIdentifier `hcl:"bucket" cty:"bucket"`

	Name string `hcl:"name" cty:"name"`
}

func (id *HCLIdentifier) CtyType() cty.Type {
	return cty.Object(map[string]cty.Type{
		"bucket": id.Bucket.CtyType(),
		"name":   cty.String,
	})
}

func (id *HCLIdentifier) ToCtyValue() (cty.Value, error) {
	return gocty.ToCtyValue(id, id.CtyType())
}

func (id *HCLIdentifier) ToIdentifier() *Identifier {
	out := &Identifier{}

	out.Bucket = id.Bucket.ToIdentifier()

	out.Name = id.Name

	return out
}

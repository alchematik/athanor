package bucket_object

import (
	"fmt"

	"strings"

	"github.com/alchematik/athanor/provider"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"
)

// Identifier is the identifier for a bucket_object.
type Identifier struct {
	// Bucket is the bucket that the object belongs to
	Bucket any

	// Name is the name of the bucket_object
	Name string
}

func (id *Identifier) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("%s", id.Bucket))

	parts = append(parts, "bucket_object", fmt.Sprintf("%s", id.Name))

	return strings.Join(parts, "/")
}

func FieldValuesToIdentifier(fieldValues []provider.FieldValue) *Identifier {
	var id Identifier

	for _, fv := range fieldValues {
		switch fv.Name {

		case "bucket":

			switch fv.Metadata["identifier_type"] {

			case "bucket":
				id.Bucket = bucket.FieldValuesToIdentifier(fv.Value.([]provider.FieldValue))

			}

		case "name":

			id.Name = fv.Value.(string)

		}
	}

	return &id
}

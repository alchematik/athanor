package bucket

import (
	"fmt"

	"strings"

	"github.com/alchematik/athanor/provider"
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

func (id *Identifier) String() string {
	var parts []string

	parts = append(parts, "aws", "v0.0.1")

	parts = append(parts, fmt.Sprintf("%s", id.Account))

	parts = append(parts, fmt.Sprintf("%s", id.Region))

	parts = append(parts, "bucket", fmt.Sprintf("%s", id.Name))

	return strings.Join(parts, "/")
}

func FieldValuesToIdentifier(fieldValues []provider.FieldValue) *Identifier {
	var id Identifier

	for _, fv := range fieldValues {
		switch fv.Name {

		case "account":

			id.Account = fv.Value.(string)

		case "region":

			id.Region = fv.Value.(string)

		case "name":

			id.Name = fv.Value.(string)

		}
	}

	return &id
}

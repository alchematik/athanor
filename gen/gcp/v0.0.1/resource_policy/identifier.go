package resource_policy

import (
	"fmt"

	"strings"

	"github.com/alchematik/athanor/provider"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"
)

// Identifier is the identifier for a resource_policy.
type Identifier struct {
	// Resource is the resource that the policy belongs to
	Resource any

	// Name is the name of the resource policy
	Name string
}

func (id *Identifier) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("%s", id.Resource))

	parts = append(parts, "resource_policy", fmt.Sprintf("%s", id.Name))

	return strings.Join(parts, "/")
}

func FieldValuesToIdentifier(fieldValues []provider.FieldValue) *Identifier {
	var id Identifier

	for _, fv := range fieldValues {
		switch fv.Name {

		case "resource":

			switch fv.Metadata["identifier_type"] {

			case "bucket":
				id.Resource = bucket.FieldValuesToIdentifier(fv.Value.([]provider.FieldValue))

			}

		case "name":

			id.Name = fv.Value.(string)

		}
	}

	return &id
}

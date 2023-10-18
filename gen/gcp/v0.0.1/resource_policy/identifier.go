package resource_policy

import (
	"fmt"

	"strings"
)

// Identifier is the identifier for a resource_policy.
type Identifier struct {
	// Resource is the resource that the policy belongs to.
	Resource any

	// Name is the name of the resource policy.
	Name string
}

func (id *Identifier) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("%s", id.Resource))

	parts = append(parts, "resource_policy", fmt.Sprintf("%s", id.Name))

	return strings.Join(parts, "/")
}

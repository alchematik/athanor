package bucket

import (
	"fmt"

	"strings"
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

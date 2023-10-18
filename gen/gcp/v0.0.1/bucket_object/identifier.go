package bucket_object

import (
	"fmt"

	"strings"

	"github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"
)

// Identifier is the identifier for a bucket_object.
type Identifier struct {
	// Bucket is the bucket that the object belongs to.
	Bucket *bucket.Identifier

	// Name is the name of the bucket_object.
	Name string
}

func (id *Identifier) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("%s", id.Bucket))

	parts = append(parts, "bucket_object", fmt.Sprintf("%s", id.Name))

	return strings.Join(parts, "/")
}

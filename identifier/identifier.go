package identifier

import (
	"github.com/zclconf/go-cty/cty"
)

type HCLIdentifier interface {
	CtyType() cty.Type
}

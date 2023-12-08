package build

import (
	"github.com/alchematik/athanor/build/value"
)

type Build struct {
	Nodes []Node
}

type Node struct {
	Value    value.Type
	Children []Node
}

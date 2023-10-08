package operation

type ResourceState string

const (
	ResourceStateExists = "exists"
)

type Operation interface {
	ForIdentifier() Identifier
	ForVersion() string
	Apply(*Resource)
}

type Identifier interface {
	String() string
}

type Resource struct {
	State      ResourceState
	Identifier Identifier
	Config     any
}

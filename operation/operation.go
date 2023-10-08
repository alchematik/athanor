package operation

type Operation interface {
	ForIdentifier() Identifier
}

type Identifier interface {
	String() string
}

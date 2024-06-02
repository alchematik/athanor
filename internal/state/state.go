package state

type State struct {
	Resources map[string]Resource
}

type Resource struct {
	Provider   Provider
	Exists     bool
	Identifier any
	Config     any
}

type Provider struct {
	Name    string
	Version string
}

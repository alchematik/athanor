package state

type State struct {
	Resources map[string]Resource
	Builds    map[string]Build
}

type Resource struct {
	Name       string
	Provider   Provider
	Exists     bool
	Identifier any
	Config     any
}

type Provider struct {
	Name    string
	Version string
}

type Build struct {
	Name string
}

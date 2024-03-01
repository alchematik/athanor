package spec

type Spec struct {
	DependencyMap map[string][]string
	Components    map[string]Component
	RuntimeConfig Value
}

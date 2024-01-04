package spec

type Spec struct {
	Providers     map[string]ValueProvider
	Resources     map[string]ValueResource
	DependencyMap map[string][]string
	Components    map[string]Component
}

package spec

type Spec struct {
	Inputs        map[string]Value
	DependencyMap map[string][]string
	Components    map[string]Component
}

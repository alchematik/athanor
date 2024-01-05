package spec

type Spec struct {
	Inputs    map[string]Value
	Providers map[string]ValueProvider
	Resources map[string]ValueResource
	// Blueprints map[string]ValueBlueprint
	// Builds     map[string]ValueBu

	DependencyMap map[string][]string
	Components    map[string]Component
}

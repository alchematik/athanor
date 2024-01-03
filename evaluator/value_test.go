package evaluator_test

import (
	"context"
	"testing"

	api "github.com/alchematik/athanor/api/resource"
	"github.com/alchematik/athanor/build/value"
	"github.com/alchematik/athanor/evaluator"
	"github.com/alchematik/athanor/state"

	"github.com/stretchr/testify/require"
)

func TestEvaluator_Value_Provider(t *testing.T) {
	testCases := map[string]struct {
		env     state.Environment
		input   value.Type
		output  state.Type
		isError bool
	}{
		"valid provider": {
			input: value.Provider{
				Identifier: value.ProviderIdentifier{
					Name:    "gcp",
					Version: "v0.0.1",
				},
			},
			output: state.Provider{
				Name:    "gcp",
				Version: "v0.0.1",
			},
		},
		"missing name": {
			input: value.Provider{
				Identifier: value.ProviderIdentifier{
					Name:    "",
					Version: "v0.0.1",
				},
			},
			output:  state.Provider{},
			isError: true,
		},
		"missing version": {
			input: value.Provider{
				Identifier: value.ProviderIdentifier{
					Name:    "gcp",
					Version: "",
				},
			},
			output:  state.Provider{},
			isError: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			eval := evaluator.Evaluator{}
			out, err := eval.Value(context.Background(), tc.env, tc.input)
			require.Equal(t, tc.output, out)
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEvaluator_Value_String(t *testing.T) {
	eval := evaluator.Evaluator{}
	env := state.Environment{}
	out, err := eval.Value(context.Background(), env, value.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t, state.String{Value: "foo"}, out)
}

func TestEvaluator_Value_Map(t *testing.T) {
	testCases := map[string]struct {
		env     state.Environment
		input   value.Type
		output  state.Type
		isError bool
	}{
		"single entry": {
			input: value.Map{
				Entries: map[string]value.Type{
					"foo": value.String{Value: "bar"},
				},
			},
			output: state.Map{
				Entries: map[string]state.Type{
					"foo": state.String{Value: "bar"},
				},
			},
		},
		"several entries": {
			input: value.Map{
				Entries: map[string]value.Type{
					"foo": value.String{Value: "bar"},
					"baz": value.String{Value: "bam"},
				},
			},
			output: state.Map{
				Entries: map[string]state.Type{
					"foo": state.String{Value: "bar"},
					"baz": state.String{Value: "bam"},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			eval := evaluator.Evaluator{}
			out, err := eval.Value(context.Background(), tc.env, tc.input)
			require.Equal(t, tc.output, out)
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEvaluator_Value_ResourceIdentifier(t *testing.T) {
	testCases := map[string]struct {
		env     state.Environment
		input   value.Type
		output  state.Type
		isError bool
	}{
		"valid": {
			input: value.ResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Value:        value.String{Value: "bucket-id"},
			},
			output: state.Identifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Value:        state.String{Value: "bucket-id"},
			},
		},
		"missing alias": {
			input: value.ResourceIdentifier{
				Alias:        "",
				ResourceType: "bucket",
				Value:        value.String{Value: "bucket-id"},
			},
			output:  state.Identifier{},
			isError: true,
		},
		"missing resource type": {
			input: value.ResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "",
				Value:        value.String{Value: "bucket-id"},
			},
			output:  state.Identifier{},
			isError: true,
		},
		"missing value": {
			input: value.ResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Value:        nil,
			},
			output:  state.Identifier{},
			isError: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			eval := evaluator.Evaluator{}
			out, err := eval.Value(context.Background(), tc.env, tc.input)
			require.Equal(t, tc.output, out)
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEvaluator_Value_ResourceRef(t *testing.T) {
	testCases := map[string]struct {
		env     state.Environment
		input   value.Type
		output  state.Type
		isError bool
	}{
		"found resource": {
			env: state.Environment{
				Resources: map[string]state.Resource{
					"my-resource": {
						Identifier: state.Identifier{
							Alias:        "my-resource",
							ResourceType: "bucket",
							Value:        state.String{Value: "foo"},
						},
					},
				},
			},
			input: value.ResourceRef{
				Alias: "my-resource",
			},
			output: state.Resource{
				Identifier: state.Identifier{
					Alias:        "my-resource",
					ResourceType: "bucket",
					Value:        state.String{Value: "foo"},
				},
			},
		},
		"resource not found": {
			env: state.Environment{
				Resources: map[string]state.Resource{},
			},
			input: value.ResourceRef{
				Alias: "my-resource",
			},
			output:  state.Resource{},
			isError: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			eval := evaluator.Evaluator{}
			out, err := eval.Value(context.Background(), tc.env, tc.input)
			require.Equal(t, tc.output, out)
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEvaluator_Value_Unresolved(t *testing.T) {
	testCases := map[string]struct {
		env     state.Environment
		input   value.Type
		output  state.Type
		isError bool
	}{
		"map entry": {
			input: value.Unresolved{
				Name: "foo",
				Object: value.Map{
					Entries: map[string]value.Type{
						"foo": value.String{Value: "bar"},
					},
				},
			},
			output: state.String{Value: "bar"},
		},
		"map entry missing": {
			input: value.Unresolved{
				Name: "foo",
				Object: value.Map{
					Entries: map[string]value.Type{},
				},
			},
			isError: true,
		},
		"resource identifier": {
			input: value.Unresolved{
				Name: "identifier",
				Object: value.Resource{
					Provider: value.Provider{
						Identifier: value.ProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
					Identifier: value.ResourceIdentifier{
						Alias:        "my-resource",
						ResourceType: "bucket",
						Value:        value.String{Value: "foo"},
					},
					Config: value.String{Value: "bar"},
				},
			},
			output: state.Identifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Value:        state.String{Value: "foo"},
			},
		},
		"resource config": {
			input: value.Unresolved{
				Name: "config",
				Object: value.Resource{
					Provider: value.Provider{
						Identifier: value.ProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
					Identifier: value.ResourceIdentifier{
						Alias:        "my-resource",
						ResourceType: "bucket",
						Value:        value.String{Value: "foo"},
					},
					Config: value.String{Value: "bar"},
				},
			},
			output: state.String{Value: "bar"},
		},
		"resource attrs": {
			input: value.Unresolved{
				Name: "attrs",
				Object: value.Resource{
					Provider: value.Provider{
						Identifier: value.ProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
					Identifier: value.ResourceIdentifier{
						Alias:        "my-resource",
						ResourceType: "bucket",
						Value:        value.String{Value: "foo"},
					},
					Config: value.String{Value: "bar"},
				},
			},
			output: state.Unknown{
				Name: "attrs",
				Object: state.ResourceRef{
					Alias: "my-resource",
				},
			},
		},
		"resource not a real field": {
			input: value.Unresolved{
				Name: "fake",
				Object: value.Resource{
					Provider: value.Provider{
						Identifier: value.ProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
					Identifier: value.ResourceIdentifier{
						Alias:        "my-resource",
						ResourceType: "bucket",
						Value:        value.String{Value: "foo"},
					},
					Config: value.String{Value: "bar"},
				},
			},
			isError: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			eval := evaluator.Evaluator{
				ResourceAPI: &api.Unresolved{},
			}
			out, err := eval.Value(context.Background(), tc.env, tc.input)
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.output, out)
		})
	}
}

func TestEvaluator_Value_Resource(t *testing.T) {
	testCases := map[string]struct {
		env     state.Environment
		input   value.Type
		output  state.Type
		isError bool
	}{
		"valid resource": {
			input: value.Resource{
				Provider: value.Provider{
					Identifier: value.ProviderIdentifier{
						Alias:   "my-provider",
						Name:    "gcp",
						Version: "v0.0.1",
					},
				},
				Identifier: value.ResourceIdentifier{
					Alias:        "my-resource",
					ResourceType: "bucket",
					Value:        value.String{Value: "foo"},
				},
				Config: value.String{Value: "bar"},
				Attrs: value.Unresolved{
					Name: "attrs",
					Object: value.ResourceRef{
						Alias: "my-resource",
					},
				},
			},
			output: state.Resource{
				Provider: state.Provider{
					Name:    "gcp",
					Version: "v0.0.1",
				},
				Identifier: state.Identifier{
					Alias:        "my-resource",
					ResourceType: "bucket",
					Value:        state.String{Value: "foo"},
				},
				Config: state.String{Value: "bar"},
				Attrs: state.Unknown{
					Name: "attrs",
					Object: state.ResourceRef{
						Alias: "my-resource",
					},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			eval := evaluator.Evaluator{
				ResourceAPI: &api.Unresolved{},
			}
			out, err := eval.Value(context.Background(), tc.env, tc.input)
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.output, out)
		})
	}
}

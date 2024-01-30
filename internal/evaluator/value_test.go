package evaluator_test

import (
	"context"
	"testing"

	api "github.com/alchematik/athanor/internal/api/resource"
	"github.com/alchematik/athanor/internal/evaluator"
	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"

	"github.com/stretchr/testify/require"
)

func TestEvaluator_Value_Provider(t *testing.T) {
	testCases := map[string]struct {
		env     state.Environment
		input   spec.Value
		output  state.Type
		isError bool
	}{
		"valid provider": {
			input: spec.ValueProvider{
				Identifier: spec.ValueProviderIdentifier{
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
			input: spec.ValueProvider{
				Identifier: spec.ValueProviderIdentifier{
					Name:    "",
					Version: "v0.0.1",
				},
			},
			output:  state.Provider{},
			isError: true,
		},
		"missing version": {
			input: spec.ValueProvider{
				Identifier: spec.ValueProviderIdentifier{
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
	out, err := eval.Value(context.Background(), env, spec.ValueString{Literal: "foo"})
	require.NoError(t, err)
	require.Equal(t, state.String{Value: "foo"}, out)
}

func TestEvaluator_Value_Map(t *testing.T) {
	testCases := map[string]struct {
		env     state.Environment
		input   spec.Value
		output  state.Type
		isError bool
	}{
		"single entry": {
			input: spec.ValueMap{
				Entries: map[string]spec.Value{
					"foo": spec.ValueString{Literal: "bar"},
				},
			},
			output: state.Map{
				Entries: map[string]state.Type{
					"foo": state.String{Value: "bar"},
				},
			},
		},
		"several entries": {
			input: spec.ValueMap{
				Entries: map[string]spec.Value{
					"foo": spec.ValueString{Literal: "bar"},
					"baz": spec.ValueString{Literal: "bam"},
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
		input   spec.Value
		output  state.Type
		isError bool
	}{
		"valid": {
			input: spec.ValueResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Literal:      spec.ValueString{Literal: "bucket-id"},
			},
			output: state.Identifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Value:        state.String{Value: "bucket-id"},
			},
		},
		"missing alias": {
			input: spec.ValueResourceIdentifier{
				Alias:        "",
				ResourceType: "bucket",
				Literal:      spec.ValueString{Literal: "bucket-id"},
			},
			output:  state.Identifier{},
			isError: true,
		},
		"missing resource type": {
			input: spec.ValueResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "",
				Literal:      spec.ValueString{Literal: "bucket-id"},
			},
			output:  state.Identifier{},
			isError: true,
		},
		"missing value": {
			input: spec.ValueResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Literal:      nil,
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
		input   spec.Value
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
			input: spec.ValueResourceRef{
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
			input: spec.ValueResourceRef{
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
		input   spec.Value
		output  state.Type
		isError bool
	}{
		"map entry": {
			input: spec.ValueUnresolved{
				Name: "foo",
				Object: spec.ValueMap{
					Entries: map[string]spec.Value{
						"foo": spec.ValueString{Literal: "bar"},
					},
				},
			},
			output: state.String{Value: "bar"},
		},
		"map entry missing": {
			input: spec.ValueUnresolved{
				Name: "foo",
				Object: spec.ValueMap{
					Entries: map[string]spec.Value{},
				},
			},
			isError: true,
		},
		"resource identifier": {
			input: spec.ValueUnresolved{
				Name: "identifier",
				Object: spec.ValueResource{
					Provider: spec.ValueProvider{
						Identifier: spec.ValueProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
					Identifier: spec.ValueResourceIdentifier{
						Alias:        "my-resource",
						ResourceType: "bucket",
						Literal:      spec.ValueString{Literal: "foo"},
					},
					Config: spec.ValueString{Literal: "bar"},
					Exists: spec.ValueBool{Literal: true},
				},
			},
			output: state.Identifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Value:        state.String{Value: "foo"},
			},
		},
		"resource config": {
			input: spec.ValueUnresolved{
				Name: "config",
				Object: spec.ValueResource{
					Exists: spec.ValueBool{Literal: true},
					Provider: spec.ValueProvider{
						Identifier: spec.ValueProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
					Identifier: spec.ValueResourceIdentifier{
						Alias:        "my-resource",
						ResourceType: "bucket",
						Literal:      spec.ValueString{Literal: "foo"},
					},
					Config: spec.ValueString{Literal: "bar"},
				},
			},
			output: state.String{Value: "bar"},
		},
		"resource attrs": {
			input: spec.ValueUnresolved{
				Name: "attrs",
				Object: spec.ValueResource{
					Exists: spec.ValueBool{Literal: true},
					Provider: spec.ValueProvider{
						Identifier: spec.ValueProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
					Identifier: spec.ValueResourceIdentifier{
						Alias:        "my-resource",
						ResourceType: "bucket",
						Literal:      spec.ValueString{Literal: "foo"},
					},
					Config: spec.ValueString{Literal: "bar"},
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
			input: spec.ValueUnresolved{
				Name: "fake",
				Object: spec.ValueResource{
					Provider: spec.ValueProvider{
						Identifier: spec.ValueProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
					Identifier: spec.ValueResourceIdentifier{
						Alias:        "my-resource",
						ResourceType: "bucket",
						Literal:      spec.ValueString{Literal: "foo"},
					},
					Config: spec.ValueString{Literal: "bar"},
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
		input   spec.Value
		output  state.Type
		isError bool
	}{
		"valid resource": {
			input: spec.ValueResource{
				Exists: spec.ValueBool{Literal: true},
				Provider: spec.ValueProvider{
					Identifier: spec.ValueProviderIdentifier{
						Alias:   "my-provider",
						Name:    "gcp",
						Version: "v0.0.1",
					},
				},
				Identifier: spec.ValueResourceIdentifier{
					Alias:        "my-resource",
					ResourceType: "bucket",
					Literal:      spec.ValueString{Literal: "foo"},
				},
				Config: spec.ValueString{Literal: "bar"},
				Attrs: spec.ValueUnresolved{
					Name: "attrs",
					Object: spec.ValueResourceRef{
						Alias: "my-resource",
					},
				},
			},
			output: state.Resource{
				Exists: state.Bool{Value: true},
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

package evaluator_test

import (
	"context"
	"testing"

	"github.com/alchematik/athanor/build/value"
	api "github.com/alchematik/athanor/internal/api/resource"
	"github.com/alchematik/athanor/internal/evaluator"
	"github.com/alchematik/athanor/state"

	"github.com/stretchr/testify/require"
)

func TestEvaluator_Value_Provider(t *testing.T) {
	testCases := map[string]struct {
		env     state.Environment
		input   value.Value
		output  state.Type
		isError bool
	}{
		"valid provider": {
			input: value.ValueProvider{
				Identifier: value.ValueProviderIdentifier{
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
			input: value.ValueProvider{
				Identifier: value.ValueProviderIdentifier{
					Name:    "",
					Version: "v0.0.1",
				},
			},
			output:  state.Provider{},
			isError: true,
		},
		"missing version": {
			input: value.ValueProvider{
				Identifier: value.ValueProviderIdentifier{
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
	out, err := eval.Value(context.Background(), env, value.ValueString{Literal: "foo"})
	require.NoError(t, err)
	require.Equal(t, state.String{Value: "foo"}, out)
}

func TestEvaluator_Value_Map(t *testing.T) {
	testCases := map[string]struct {
		env     state.Environment
		input   value.Value
		output  state.Type
		isError bool
	}{
		"single entry": {
			input: value.ValueMap{
				Entries: map[string]value.Value{
					"foo": value.ValueString{Literal: "bar"},
				},
			},
			output: state.Map{
				Entries: map[string]state.Type{
					"foo": state.String{Value: "bar"},
				},
			},
		},
		"several entries": {
			input: value.ValueMap{
				Entries: map[string]value.Value{
					"foo": value.ValueString{Literal: "bar"},
					"baz": value.ValueString{Literal: "bam"},
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
		input   value.Value
		output  state.Type
		isError bool
	}{
		"valid": {
			input: value.ValueResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Literal:      value.ValueString{Literal: "bucket-id"},
			},
			output: state.Identifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Value:        state.String{Value: "bucket-id"},
			},
		},
		"missing alias": {
			input: value.ValueResourceIdentifier{
				Alias:        "",
				ResourceType: "bucket",
				Literal:      value.ValueString{Literal: "bucket-id"},
			},
			output:  state.Identifier{},
			isError: true,
		},
		"missing resource type": {
			input: value.ValueResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "",
				Literal:      value.ValueString{Literal: "bucket-id"},
			},
			output:  state.Identifier{},
			isError: true,
		},
		"missing value": {
			input: value.ValueResourceIdentifier{
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
		input   value.Value
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
			input: value.ValueResourceRef{
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
			input: value.ValueResourceRef{
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
		input   value.Value
		output  state.Type
		isError bool
	}{
		"map entry": {
			input: value.ValueUnresolved{
				Name: "foo",
				Object: value.ValueMap{
					Entries: map[string]value.Value{
						"foo": value.ValueString{Literal: "bar"},
					},
				},
			},
			output: state.String{Value: "bar"},
		},
		"map entry missing": {
			input: value.ValueUnresolved{
				Name: "foo",
				Object: value.ValueMap{
					Entries: map[string]value.Value{},
				},
			},
			isError: true,
		},
		"resource identifier": {
			input: value.ValueUnresolved{
				Name: "identifier",
				Object: value.ValueResource{
					Provider: value.ValueProvider{
						Identifier: value.ValueProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
					Identifier: value.ValueResourceIdentifier{
						Alias:        "my-resource",
						ResourceType: "bucket",
						Literal:      value.ValueString{Literal: "foo"},
					},
					Config: value.ValueString{Literal: "bar"},
					Exists: value.ValueBool{Literal: true},
				},
			},
			output: state.Identifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Value:        state.String{Value: "foo"},
			},
		},
		"resource config": {
			input: value.ValueUnresolved{
				Name: "config",
				Object: value.ValueResource{
					Exists: value.ValueBool{Literal: true},
					Provider: value.ValueProvider{
						Identifier: value.ValueProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
					Identifier: value.ValueResourceIdentifier{
						Alias:        "my-resource",
						ResourceType: "bucket",
						Literal:      value.ValueString{Literal: "foo"},
					},
					Config: value.ValueString{Literal: "bar"},
				},
			},
			output: state.String{Value: "bar"},
		},
		"resource attrs": {
			input: value.ValueUnresolved{
				Name: "attrs",
				Object: value.ValueResource{
					Exists: value.ValueBool{Literal: true},
					Provider: value.ValueProvider{
						Identifier: value.ValueProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
					Identifier: value.ValueResourceIdentifier{
						Alias:        "my-resource",
						ResourceType: "bucket",
						Literal:      value.ValueString{Literal: "foo"},
					},
					Config: value.ValueString{Literal: "bar"},
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
			input: value.ValueUnresolved{
				Name: "fake",
				Object: value.ValueResource{
					Provider: value.ValueProvider{
						Identifier: value.ValueProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
					Identifier: value.ValueResourceIdentifier{
						Alias:        "my-resource",
						ResourceType: "bucket",
						Literal:      value.ValueString{Literal: "foo"},
					},
					Config: value.ValueString{Literal: "bar"},
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
		input   value.Value
		output  state.Type
		isError bool
	}{
		"valid resource": {
			input: value.ValueResource{
				Exists: value.ValueBool{Literal: true},
				Provider: value.ValueProvider{
					Identifier: value.ValueProviderIdentifier{
						Alias:   "my-provider",
						Name:    "gcp",
						Version: "v0.0.1",
					},
				},
				Identifier: value.ValueResourceIdentifier{
					Alias:        "my-resource",
					ResourceType: "bucket",
					Literal:      value.ValueString{Literal: "foo"},
				},
				Config: value.ValueString{Literal: "bar"},
				Attrs: value.ValueUnresolved{
					Name: "attrs",
					Object: value.ValueResourceRef{
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

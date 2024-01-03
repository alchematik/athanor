package evaluator_test

import (
	"context"
	"testing"

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

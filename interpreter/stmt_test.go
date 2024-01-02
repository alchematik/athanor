package interpreter_test

import (
	"context"
	"testing"

	"github.com/alchematik/athanor/blueprint/expr"
	"github.com/alchematik/athanor/blueprint/stmt"
	"github.com/alchematik/athanor/build/value"
	"github.com/alchematik/athanor/interpreter"

	"github.com/stretchr/testify/require"
)

func TestInterpreter_Stmt_Provider(t *testing.T) {
	testCases := map[string]struct {
		env         interpreter.Environment
		expectedEnv interpreter.Environment
		stmt        stmt.Type
		isError     bool
	}{
		"new provider": {
			env: interpreter.NewEnvironment(),
			stmt: stmt.Provider{
				Identifier: expr.ProviderIdentifier{
					Alias:   "my-provider",
					Name:    expr.String{Value: "gcp"},
					Version: expr.String{Value: "v0.0.1"},
				},
			},
			expectedEnv: interpreter.Environment{
				Providers: map[string]value.Provider{
					"my-provider": {
						Identifier: value.ProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
				},
				Resources:     map[string]value.Resource{},
				DependencyMap: map[string][]string{},
			},
		},
		"value not ProviderIdentifier": {
			env: interpreter.NewEnvironment(),
			stmt: stmt.Provider{
				Identifier: expr.ResourceIdentifier{
					Alias: "my-provider",
				},
			},
			expectedEnv: interpreter.NewEnvironment(),
			isError:     true,
		},
		"invalid ProviderIdentifier": {
			env: interpreter.NewEnvironment(),
			stmt: stmt.Provider{
				Identifier: expr.ProviderIdentifier{
					Alias:   "",
					Name:    expr.String{Value: "gcp"},
					Version: expr.String{Value: "v0.0.1"},
				},
			},
			expectedEnv: interpreter.NewEnvironment(),
			isError:     true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			in := interpreter.Interpreter{}
			err := in.Stmt(context.Background(), tc.env, tc.stmt)
			if tc.isError {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
			}

			require.Equal(t, tc.expectedEnv, tc.env)
		})
	}
}

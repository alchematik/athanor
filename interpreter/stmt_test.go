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
		env         value.Build
		expectedEnv value.Build
		stmt        stmt.Type
		isError     bool
	}{
		"new provider": {
			env: value.Build{
				Providers:     map[string]value.Provider{},
				Resources:     map[string]value.Resource{},
				DependencyMap: map[string][]string{},
			},
			stmt: stmt.Provider{
				Expr: expr.Provider{
					Identifier: expr.ProviderIdentifier{
						Alias:   "my-provider",
						Name:    expr.String{Value: "gcp"},
						Version: expr.String{Value: "v0.0.1"},
					},
				},
			},
			expectedEnv: value.Build{
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
		"value not provider": {
			env: value.Build{
				Providers:     map[string]value.Provider{},
				Resources:     map[string]value.Resource{},
				DependencyMap: map[string][]string{},
			},
			stmt: stmt.Provider{
				Expr: expr.ProviderIdentifier{
					Alias:   "my-provider",
					Name:    expr.String{Value: "gcp"},
					Version: expr.String{Value: "v0.0.1"},
				},
			},
			expectedEnv: value.Build{
				Providers:     map[string]value.Provider{},
				Resources:     map[string]value.Resource{},
				DependencyMap: map[string][]string{},
			},
			isError: true,
		},
		"invalid Provider": {
			env: value.Build{
				Providers:     map[string]value.Provider{},
				Resources:     map[string]value.Resource{},
				DependencyMap: map[string][]string{},
			},
			stmt: stmt.Provider{
				Expr: expr.Provider{
					Identifier: expr.ProviderIdentifier{
						Alias:   "",
						Name:    expr.String{Value: "gcp"},
						Version: expr.String{Value: "v0.0.1"},
					},
				},
			},
			expectedEnv: value.Build{
				Providers:     map[string]value.Provider{},
				Resources:     map[string]value.Resource{},
				DependencyMap: map[string][]string{},
			},
			isError: true,
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

func TestInterpreter_Stmt_Resource(t *testing.T) {
	testCases := map[string]struct {
		env         value.Build
		expectedEnv value.Build
		stmt        stmt.Type
		isError     bool
	}{
		"new resource": {
			env: value.Build{
				Providers:     map[string]value.Provider{},
				Resources:     map[string]value.Resource{},
				DependencyMap: map[string][]string{},
			},
			stmt: stmt.Resource{
				Expr: expr.Resource{
					Identifier: expr.ResourceIdentifier{
						Alias:        "my-resource",
						ResourceType: "bucket",
						Value:        expr.String{Value: "foo"},
					},
					Provider: expr.Provider{
						Identifier: expr.ProviderIdentifier{
							Alias:   "my-provider",
							Name:    expr.String{Value: "gcp"},
							Version: expr.String{Value: "v0.0.1"},
						},
					},
					Config: expr.String{Value: "bar"},
				},
			},
			expectedEnv: value.Build{
				Providers: map[string]value.Provider{
					"my-provider": {
						Identifier: value.ProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
				},
				Resources: map[string]value.Resource{
					"my-resource": {
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
				},
				DependencyMap: map[string][]string{
					"my-resource": nil,
				},
			},
		},
		"value not Resource": {
			env: value.Build{
				Providers:     map[string]value.Provider{},
				Resources:     map[string]value.Resource{},
				DependencyMap: map[string][]string{},
			},
			stmt: stmt.Resource{
				Expr: expr.Provider{
					Identifier: expr.ProviderIdentifier{
						Alias:   "my-provider",
						Name:    expr.String{Value: "gcp"},
						Version: expr.String{Value: "v0.0.1"},
					},
				},
			},
			expectedEnv: value.Build{
				Providers:     map[string]value.Provider{},
				Resources:     map[string]value.Resource{},
				DependencyMap: map[string][]string{},
			},
			isError: true,
		},
		"invalid Resource": {
			env: value.Build{
				Providers:     map[string]value.Provider{},
				Resources:     map[string]value.Resource{},
				DependencyMap: map[string][]string{},
			},
			stmt: stmt.Resource{
				Expr: expr.Resource{
					Identifier: expr.ResourceIdentifier{
						Alias:        "", // Invalid because alias shouldn't be an empty string.
						ResourceType: "bucket",
						Value:        expr.String{Value: "foo"},
					},
					Provider: expr.Provider{
						Identifier: expr.ProviderIdentifier{
							Alias:   "my-provider",
							Name:    expr.String{Value: "gcp"},
							Version: expr.String{Value: "v0.0.1"},
						},
					},
					Config: expr.String{Value: "bar"},
				},
			},
			expectedEnv: value.Build{
				Providers:     map[string]value.Provider{},
				Resources:     map[string]value.Resource{},
				DependencyMap: map[string][]string{},
			},
			isError: true,
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

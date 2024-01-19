package interpreter_test

import (
	"context"
	"testing"

	"github.com/alchematik/athanor/internal/blueprint/expr"
	"github.com/alchematik/athanor/internal/blueprint/stmt"
	"github.com/alchematik/athanor/internal/build"
	"github.com/alchematik/athanor/internal/build/component"
	"github.com/alchematik/athanor/internal/build/value"
	"github.com/alchematik/athanor/internal/interpreter"

	"github.com/stretchr/testify/require"
)

func TestInterpreter_Stmt_Provider(t *testing.T) {
	testCases := map[string]struct {
		env         build.Spec
		expectedEnv build.Spec
		stmt        stmt.Type
		isError     bool
	}{
		"new provider": {
			env: build.Spec{
				Providers:     map[string]value.ValueProvider{},
				Resources:     map[string]value.ValueResource{},
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
			expectedEnv: build.Spec{
				Providers: map[string]value.ValueProvider{
					"my-provider": {
						Identifier: value.ValueProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
				},
				Resources:     map[string]value.ValueResource{},
				DependencyMap: map[string][]string{},
			},
		},
		"value not provider": {
			env: build.Spec{
				Providers:     map[string]value.ValueProvider{},
				Resources:     map[string]value.ValueResource{},
				DependencyMap: map[string][]string{},
			},
			stmt: stmt.Provider{
				Expr: expr.ProviderIdentifier{
					Alias:   "my-provider",
					Name:    expr.String{Value: "gcp"},
					Version: expr.String{Value: "v0.0.1"},
				},
			},
			expectedEnv: build.Spec{
				Providers:     map[string]value.ValueProvider{},
				Resources:     map[string]value.ValueResource{},
				DependencyMap: map[string][]string{},
			},
			isError: true,
		},
		"invalid Provider": {
			env: build.Spec{
				Providers:     map[string]value.ValueProvider{},
				Resources:     map[string]value.ValueResource{},
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
			expectedEnv: build.Spec{
				Providers:     map[string]value.ValueProvider{},
				Resources:     map[string]value.ValueResource{},
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
		env         build.Spec
		expectedEnv build.Spec
		stmt        stmt.Type
		isError     bool
	}{
		"new resource": {
			env: build.Spec{
				Providers:     map[string]value.ValueProvider{},
				Resources:     map[string]value.ValueResource{},
				DependencyMap: map[string][]string{},
				Components:    map[string]component.Component{},
			},
			stmt: stmt.Resource{
				Expr: expr.Resource{
					Exists: expr.Bool{Value: true},
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
			expectedEnv: build.Spec{
				Providers: map[string]value.ValueProvider{
					"my-provider": {
						Identifier: value.ValueProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
				},
				Resources: map[string]value.ValueResource{
					"my-resource": {
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
				},
				DependencyMap: map[string][]string{
					"my-resource": nil,
				},
				Components: map[string]component.Component{
					"my-resource": component.ComponentResource{
						Value: value.ValueResource{
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
					},
				},
			},
		},
		"value not Resource": {
			env: build.Spec{
				Providers:     map[string]value.ValueProvider{},
				Resources:     map[string]value.ValueResource{},
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
			expectedEnv: build.Spec{
				Providers:     map[string]value.ValueProvider{},
				Resources:     map[string]value.ValueResource{},
				DependencyMap: map[string][]string{},
			},
			isError: true,
		},
		"invalid Resource": {
			env: build.Spec{
				Providers:     map[string]value.ValueProvider{},
				Resources:     map[string]value.ValueResource{},
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
			expectedEnv: build.Spec{
				Providers:     map[string]value.ValueProvider{},
				Resources:     map[string]value.ValueResource{},
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

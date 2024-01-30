package interpreter_test

import (
	"context"
	"testing"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/interpreter"
	"github.com/alchematik/athanor/internal/spec"

	"github.com/stretchr/testify/require"
)

func TestInterpreter_Stmt_Provider(t *testing.T) {
	testCases := map[string]struct {
		env         spec.Spec
		expectedEnv spec.Spec
		stmt        spec.Value
		isError     bool
	}{
		"new provider": {
			env: spec.Spec{
				Providers:     map[string]spec.ValueProvider{},
				Resources:     map[string]spec.ValueResource{},
				DependencyMap: map[string][]string{},
			},
			stmt: spec.Provider{
				Expr: ast.Provider{
					Identifier: ast.ProviderIdentifier{
						Alias:   "my-provider",
						Name:    ast.String{Value: "gcp"},
						Version: ast.String{Value: "v0.0.1"},
					},
				},
			},
			expectedEnv: spec.Spec{
				Providers: map[string]spec.ValueProvider{
					"my-provider": {
						Identifier: spec.ValueProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
				},
				Resources:     map[string]spec.ValueResource{},
				DependencyMap: map[string][]string{},
			},
		},
		"value not provider": {
			env: spec.Spec{
				Providers:     map[string]spec.ValueProvider{},
				Resources:     map[string]spec.ValueResource{},
				DependencyMap: map[string][]string{},
			},
			stmt: spec.Provider{
				Expr: ast.ProviderIdentifier{
					Alias:   "my-provider",
					Name:    ast.String{Value: "gcp"},
					Version: ast.String{Value: "v0.0.1"},
				},
			},
			expectedEnv: spec.Spec{
				Providers:     map[string]spec.ValueProvider{},
				Resources:     map[string]spec.ValueResource{},
				DependencyMap: map[string][]string{},
			},
			isError: true,
		},
		"invalid Provider": {
			env: spec.Spec{
				Providers:     map[string]spec.ValueProvider{},
				Resources:     map[string]spec.ValueResource{},
				DependencyMap: map[string][]string{},
			},
			stmt: spec.Provider{
				Expr: ast.Provider{
					Identifier: ast.ProviderIdentifier{
						Alias:   "",
						Name:    ast.String{Value: "gcp"},
						Version: ast.String{Value: "v0.0.1"},
					},
				},
			},
			expectedEnv: spec.Spec{
				Providers:     map[string]spec.ValueProvider{},
				Resources:     map[string]spec.ValueResource{},
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
		env         spec.Spec
		expectedEnv spec.Spec
		stmt        spec.Value
		isError     bool
	}{
		"new resource": {
			env: spec.Spec{
				Providers:     map[string]spec.ValueProvider{},
				Resources:     map[string]spec.ValueResource{},
				DependencyMap: map[string][]string{},
				Components:    map[string]component.Component{},
			},
			stmt: spec.Resource{
				Expr: ast.Resource{
					Exists: ast.Bool{Value: true},
					Identifier: ast.ResourceIdentifier{
						Alias:        "my-resource",
						ResourceType: "bucket",
						Value:        ast.String{Value: "foo"},
					},
					Provider: ast.Provider{
						Identifier: ast.ProviderIdentifier{
							Alias:   "my-provider",
							Name:    ast.String{Value: "gcp"},
							Version: ast.String{Value: "v0.0.1"},
						},
					},
					Config: ast.String{Value: "bar"},
				},
			},
			expectedEnv: spec.Spec{
				Providers: map[string]spec.ValueProvider{
					"my-provider": {
						Identifier: spec.ValueProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
				},
				Resources: map[string]spec.ValueResource{
					"my-resource": {
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
				},
				DependencyMap: map[string][]string{
					"my-resource": nil,
				},
				Components: map[string]component.Component{
					"my-resource": component.ComponentResource{
						Value: spec.ValueResource{
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
					},
				},
			},
		},
		"value not Resource": {
			env: spec.Spec{
				Providers:     map[string]spec.ValueProvider{},
				Resources:     map[string]spec.ValueResource{},
				DependencyMap: map[string][]string{},
			},
			stmt: spec.Resource{
				Expr: ast.Provider{
					Identifier: ast.ProviderIdentifier{
						Alias:   "my-provider",
						Name:    ast.String{Value: "gcp"},
						Version: ast.String{Value: "v0.0.1"},
					},
				},
			},
			expectedEnv: spec.Spec{
				Providers:     map[string]spec.ValueProvider{},
				Resources:     map[string]spec.ValueResource{},
				DependencyMap: map[string][]string{},
			},
			isError: true,
		},
		"invalid Resource": {
			env: spec.Spec{
				Providers:     map[string]spec.ValueProvider{},
				Resources:     map[string]spec.ValueResource{},
				DependencyMap: map[string][]string{},
			},
			stmt: spec.Resource{
				Expr: ast.Resource{
					Identifier: ast.ResourceIdentifier{
						Alias:        "", // Invalid because alias shouldn't be an empty string.
						ResourceType: "bucket",
						Value:        ast.String{Value: "foo"},
					},
					Provider: ast.Provider{
						Identifier: ast.ProviderIdentifier{
							Alias:   "my-provider",
							Name:    ast.String{Value: "gcp"},
							Version: ast.String{Value: "v0.0.1"},
						},
					},
					Config: ast.String{Value: "bar"},
				},
			},
			expectedEnv: spec.Spec{
				Providers:     map[string]spec.ValueProvider{},
				Resources:     map[string]spec.ValueResource{},
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

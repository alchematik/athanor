package interpreter_test

import (
	"context"
	"testing"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/interpreter"
	"github.com/alchematik/athanor/internal/spec"

	"github.com/stretchr/testify/require"
)

func TestInterpreter_Expr_Map(t *testing.T) {
	testCases := map[string]struct {
		build    spec.Spec
		expr     ast.Type
		out      spec.Value
		children []string
		isError  bool
	}{
		"one entry": {
			build: spec.Spec{},
			expr: ast.Map{
				Entries: map[string]ast.Type{
					"foo": ast.String{Value: "bar"},
				},
			},
			out: spec.ValueMap{
				Entries: map[string]spec.Value{
					"foo": spec.ValueString{Literal: "bar"},
				},
			},
		},
		"several entries": {
			build: spec.Spec{},
			expr: ast.Map{
				Entries: map[string]ast.Type{
					"foo": ast.String{Value: "bar"},
					"bam": ast.String{Value: "baz"},
				},
			},
			out: spec.ValueMap{
				Entries: map[string]spec.Value{
					"foo": spec.ValueString{Literal: "bar"},
					"bam": spec.ValueString{Literal: "baz"},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			in := interpreter.Interpreter{}
			out, children, err := in.Expr(context.Background(), tc.build, tc.expr)

			require.Equal(t, tc.out, out)
			require.Equal(t, tc.children, children)
			if tc.isError {
				require.Error(t, err)
			}
		})
	}
}

func TestInterpreter_Expr_Get(t *testing.T) {
	testCases := map[string]struct {
		build    spec.Spec
		expr     ast.Type
		out      spec.Value
		children []string
		isError  bool
	}{
		"map, string value entry present": {
			build: spec.Spec{},
			expr: ast.Get{
				Name: "foo",
				Object: ast.Map{
					Entries: map[string]ast.Type{
						"foo": ast.String{Value: "bar"},
					},
				},
			},
			out: spec.ValueString{Literal: "bar"},
		},
		"map, entry missing": {
			build: spec.Spec{},
			expr: ast.Get{
				Name: "foo",
				Object: ast.Map{
					Entries: map[string]ast.Type{},
				},
			},
			isError: true,
		},
		"resource, identifier": {
			build: spec.Spec{
				Resources: map[string]spec.ValueResource{
					"my-resource": {
						Identifier: spec.ValueResourceIdentifier{
							Alias:        "my-resource",
							ResourceType: "bucket",
							Literal:      spec.ValueString{Literal: "id"},
						},
					},
				},
			},
			expr: ast.Get{
				Name: "identifier",
				Object: ast.GetResource{
					Alias: "my-resource",
				},
			},
			out: spec.ValueResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Literal:      spec.ValueString{Literal: "id"},
			},
			children: []string{"my-resource"},
		},
		"resource, config": {
			build: spec.Spec{
				Resources: map[string]spec.ValueResource{
					"my-resource": {
						Config: spec.ValueString{Literal: "config-val"},
					},
				},
			},
			expr: ast.Get{
				Name: "config",
				Object: ast.GetResource{
					Alias: "my-resource",
				},
			},
			out:      spec.ValueString{Literal: "config-val"},
			children: []string{"my-resource"},
		},
		"resource, attrs, unresolved": {
			build: spec.Spec{
				Resources: map[string]spec.ValueResource{
					"my-resource": {
						Attrs: spec.ValueUnresolved{
							Name: "attrs",
							Object: spec.ValueResourceRef{
								Alias: "my-resource",
							},
						},
					},
				},
			},
			expr: ast.Get{
				Name: "attrs",
				Object: ast.GetResource{
					Alias: "my-resource",
				},
			},
			out: spec.ValueUnresolved{
				Name: "attrs",
				Object: spec.ValueResourceRef{
					Alias: "my-resource",
				},
			},
			children: []string{"my-resource"},
		},
		"resource, attrs": {
			build: spec.Spec{
				Resources: map[string]spec.ValueResource{
					"my-resource": {
						Attrs: spec.ValueString{Literal: "foo"},
					},
				},
			},
			expr: ast.Get{
				Name: "attrs",
				Object: ast.GetResource{
					Alias: "my-resource",
				},
			},
			out:      spec.ValueString{Literal: "foo"},
			children: []string{"my-resource"},
		},
		"unresolved": {
			build: spec.Spec{
				Resources: map[string]spec.ValueResource{
					"my-resource": {
						Attrs: spec.ValueUnresolved{
							Name: "attrs",
							Object: spec.ValueResourceRef{
								Alias: "my-resource",
							},
						},
					},
				},
			},
			expr: ast.Get{
				Name: "foo",
				Object: ast.Get{
					Name: "attrs",
					Object: ast.GetResource{
						Alias: "my-resource",
					},
				},
			},
			isError: true,
		},
		"unsupported type": {
			expr: ast.Get{
				Name:   "foo",
				Object: ast.String{Value: "foo"},
			},
			isError: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			in := interpreter.Interpreter{}
			out, children, err := in.Expr(context.Background(), tc.build, tc.expr)

			require.Equal(t, tc.out, out)
			require.Equal(t, tc.children, children)
			if tc.isError {
				require.Error(t, err)
			}
		})
	}
}

func TestInterpreter_Expr_IOGet(t *testing.T) {
	testCases := map[string]struct {
		build    spec.Spec
		expr     ast.Type
		out      spec.Value
		children []string
		isError  bool
	}{
		"unresolved": {
			build: spec.Spec{
				Resources: map[string]spec.ValueResource{
					"my-resource": {
						Attrs: spec.ValueUnresolved{
							Name: "attrs",
							Object: spec.ValueResourceRef{
								Alias: "my-resource",
							},
						},
					},
				},
			},
			expr: ast.IOGet{
				Name: "foo",
				Object: ast.Get{
					Name: "attrs",
					Object: ast.GetResource{
						Alias: "my-resource",
					},
				},
			},
			out: spec.ValueUnresolved{
				Name: "foo",
				Object: spec.ValueUnresolved{
					Name: "attrs",
					Object: spec.ValueResourceRef{
						Alias: "my-resource",
					},
				},
			},
			children: []string{"my-resource"},
		},
		"map": {
			expr: ast.IOGet{
				Name: "foo",
				Object: ast.Map{
					Entries: map[string]ast.Type{
						"foo": ast.String{Value: "val"},
					},
				},
			},
			out:     spec.ValueUnresolved{},
			isError: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			in := interpreter.Interpreter{}
			out, children, err := in.Expr(context.Background(), tc.build, tc.expr)

			require.Equal(t, tc.out, out)
			require.Equal(t, tc.children, children)
			if tc.isError {
				require.Error(t, err)
			}
		})
	}
}

func TestInterpreter_Expr_String(t *testing.T) {
	testCases := map[string]struct {
		build    spec.Spec
		expr     ast.Type
		out      spec.Value
		children []string
		isError  bool
	}{
		"valid": {
			build: spec.Spec{},
			expr:  ast.String{Value: "hello world"},
			out:   spec.ValueString{Literal: "hello world"},
		},
		// IOGet
		"io get: unresolved": {
			build: spec.Spec{
				Resources: map[string]spec.ValueResource{
					"my-resource": {
						Attrs: spec.ValueUnresolved{
							Name: "attrs",
							Object: spec.ValueResourceRef{
								Alias: "my-resource",
							},
						},
					},
				},
			},
			expr: ast.IOGet{
				Name: "foo",
				Object: ast.Get{
					Name: "attrs",
					Object: ast.GetResource{
						Alias: "my-resource",
					},
				},
			},
			out: spec.ValueUnresolved{
				Name: "foo",
				Object: spec.ValueUnresolved{
					Name: "attrs",
					Object: spec.ValueResourceRef{
						Alias: "my-resource",
					},
				},
			},
			children: []string{"my-resource"},
		},
		"io get: map": {
			expr: ast.IOGet{
				Name: "foo",
				Object: ast.Map{
					Entries: map[string]ast.Type{
						"foo": ast.String{Value: "val"},
					},
				},
			},
			out:     spec.ValueUnresolved{},
			isError: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			in := interpreter.Interpreter{}
			out, children, err := in.Expr(context.Background(), tc.build, tc.expr)

			require.Equal(t, tc.out, out)
			require.Equal(t, tc.children, children)
			if tc.isError {
				require.Error(t, err)
			}
		})
	}
}

func TestInterpreter_Expr_ProviderIdentifier(t *testing.T) {
	testCases := map[string]struct {
		build    spec.Spec
		expr     ast.Type
		out      spec.Value
		children []string
		isError  bool
	}{
		"provider identifier: valid": {
			expr: ast.ProviderIdentifier{
				Alias:   "my-provider",
				Name:    ast.String{Value: "gcp"},
				Version: ast.String{Value: "v0.0.1"},
			},
			out: spec.ValueProviderIdentifier{
				Alias:   "my-provider",
				Name:    "gcp",
				Version: "v0.0.1",
			},
		},
		"provider identifier: get name": {
			expr: ast.ProviderIdentifier{
				Alias: "my-provider",
				Name: ast.Get{
					Name: "name",
					Object: ast.Map{
						Entries: map[string]ast.Type{
							"name": ast.String{Value: "gcp"},
						},
					},
				},
				Version: ast.String{Value: "v0.0.1"},
			},
			out: spec.ValueProviderIdentifier{
				Alias:   "my-provider",
				Name:    "gcp",
				Version: "v0.0.1",
			},
		},
		"provider identifier: get version": {
			expr: ast.ProviderIdentifier{
				Alias: "my-provider",
				Name:  ast.String{Value: "gcp"},
				Version: ast.Get{
					Name: "version",
					Object: ast.Map{
						Entries: map[string]ast.Type{
							"version": ast.String{Value: "v0.0.1"},
						},
					},
				},
			},
			out: spec.ValueProviderIdentifier{
				Alias:   "my-provider",
				Name:    "gcp",
				Version: "v0.0.1",
			},
		},
		"provider identifier: missing alias": {
			expr: ast.ProviderIdentifier{
				Alias:   "",
				Name:    ast.String{Value: "gcp"},
				Version: ast.String{Value: "v0.0.1"},
			},
			out:     spec.ValueProviderIdentifier{},
			isError: true,
		},
		"provider identifier: name is not string": {
			expr: ast.ProviderIdentifier{
				Alias: "my-provider",
				Name: ast.Map{
					Entries: map[string]ast.Type{
						"name": ast.String{Value: "gcp"},
					},
				},
				Version: ast.String{Value: "v0.0.1"},
			},
			out:     spec.ValueProviderIdentifier{},
			isError: true,
		},
		"provider identifier: version is not string": {
			expr: ast.ProviderIdentifier{
				Alias: "my-provider",
				Name:  ast.String{Value: "gcp"},
				Version: ast.Map{
					Entries: map[string]ast.Type{
						"name": ast.String{Value: "gcp"},
					},
				},
			},
			out:     spec.ValueProviderIdentifier{},
			isError: true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			in := interpreter.Interpreter{}
			out, children, err := in.Expr(context.Background(), tc.build, tc.expr)

			require.Equal(t, tc.out, out)
			require.Equal(t, tc.children, children)
			if tc.isError {
				require.Error(t, err)
			}
		})
	}
}

func TestInterpreter_Expr_ResourceIdentifier(t *testing.T) {
	testCases := map[string]struct {
		build    spec.Spec
		expr     ast.Type
		out      spec.Value
		children []string
		isError  bool
	}{
		"valid": {
			expr: ast.ResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Value:        ast.String{Value: "foo"},
			},
			out: spec.ValueResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Literal:      spec.ValueString{Literal: "foo"},
			},
			children: []string{"my-resource"},
		},
		"missing alias": {
			expr: ast.ResourceIdentifier{
				Alias:        "",
				ResourceType: "bucket",
				Value:        ast.String{Value: "foo"},
			},
			out:     spec.ValueResourceIdentifier{},
			isError: true,
		},
		"missing resource type": {
			expr: ast.ResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "",
				Value:        ast.String{Value: "foo"},
			},
			out:     spec.ValueResourceIdentifier{},
			isError: true,
		},
		"missing value": {
			expr: ast.ResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "",
				Value:        nil,
			},
			out:     spec.ValueResourceIdentifier{},
			isError: true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			in := interpreter.Interpreter{}
			out, children, err := in.Expr(context.Background(), tc.build, tc.expr)

			require.Equal(t, tc.out, out)
			require.Equal(t, tc.children, children)
			if tc.isError {
				require.Error(t, err)
			}
		})
	}
}

func TestInterpreter_Expr_GetProvider(t *testing.T) {
	testCases := map[string]struct {
		build    spec.Spec
		expr     ast.Type
		out      spec.Value
		children []string
		isError  bool
	}{
		"present": {
			build: spec.Spec{
				Providers: map[string]spec.ValueProvider{
					"my-provider": {
						Identifier: spec.ValueProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
				},
			},
			expr: ast.GetProvider{
				Alias: "my-provider",
			},
			out: spec.ValueProvider{
				Identifier: spec.ValueProviderIdentifier{
					Alias:   "my-provider",
					Name:    "gcp",
					Version: "v0.0.1",
				},
			},
		},
		"not present": {
			build: spec.Spec{
				Providers: map[string]spec.ValueProvider{},
			},
			expr: ast.GetProvider{
				Alias: "my-provider",
			},
			out: spec.ValueProvider{},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			in := interpreter.Interpreter{}
			out, children, err := in.Expr(context.Background(), tc.build, tc.expr)

			require.Equal(t, tc.out, out)
			require.Equal(t, tc.children, children)
			if tc.isError {
				require.Error(t, err)
			}
		})
	}
}

func TestInterpreter_Expr_GetResource(t *testing.T) {
	testCases := map[string]struct {
		build    spec.Spec
		expr     ast.Type
		out      spec.Value
		children []string
		isError  bool
	}{
		"present": {
			build: spec.Spec{
				Resources: map[string]spec.ValueResource{
					"my-resource": {
						Identifier: spec.ValueResourceIdentifier{
							Alias:        "my-resource",
							ResourceType: "bucket",
							Literal:      spec.ValueString{Literal: "foo"},
						},
					},
				},
			},
			expr: ast.GetResource{
				Alias: "my-resource",
			},
			out: spec.ValueResource{
				Identifier: spec.ValueResourceIdentifier{
					Alias:        "my-resource",
					ResourceType: "bucket",
					Literal:      spec.ValueString{Literal: "foo"},
				},
			},
			children: []string{"my-resource"},
		},
		"not present": {
			expr: ast.GetResource{
				Alias: "my-resource",
			},
			out: spec.ValueResource{},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			in := interpreter.Interpreter{}
			out, children, err := in.Expr(context.Background(), tc.build, tc.expr)

			require.Equal(t, tc.out, out)
			require.Equal(t, tc.children, children)
			if tc.isError {
				require.Error(t, err)
			}
		})
	}
}

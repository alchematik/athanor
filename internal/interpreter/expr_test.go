package interpreter_test

import (
	"context"
	"testing"

	"github.com/alchematik/athanor/internal/blueprint/expr"
	"github.com/alchematik/athanor/internal/build"
	"github.com/alchematik/athanor/internal/build/value"
	"github.com/alchematik/athanor/internal/interpreter"

	"github.com/stretchr/testify/require"
)

func TestInterpreter_Expr_Map(t *testing.T) {
	testCases := map[string]struct {
		build    build.Spec
		expr     expr.Type
		out      value.Value
		children []string
		isError  bool
	}{
		"one entry": {
			build: build.Spec{},
			expr: expr.Map{
				Entries: map[string]expr.Type{
					"foo": expr.String{Value: "bar"},
				},
			},
			out: value.ValueMap{
				Entries: map[string]value.Value{
					"foo": value.ValueString{Literal: "bar"},
				},
			},
		},
		"several entries": {
			build: build.Spec{},
			expr: expr.Map{
				Entries: map[string]expr.Type{
					"foo": expr.String{Value: "bar"},
					"bam": expr.String{Value: "baz"},
				},
			},
			out: value.ValueMap{
				Entries: map[string]value.Value{
					"foo": value.ValueString{Literal: "bar"},
					"bam": value.ValueString{Literal: "baz"},
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
		build    build.Spec
		expr     expr.Type
		out      value.Value
		children []string
		isError  bool
	}{
		"map, string value entry present": {
			build: build.Spec{},
			expr: expr.Get{
				Name: "foo",
				Object: expr.Map{
					Entries: map[string]expr.Type{
						"foo": expr.String{Value: "bar"},
					},
				},
			},
			out: value.ValueString{Literal: "bar"},
		},
		"map, entry missing": {
			build: build.Spec{},
			expr: expr.Get{
				Name: "foo",
				Object: expr.Map{
					Entries: map[string]expr.Type{},
				},
			},
			isError: true,
		},
		"resource, identifier": {
			build: build.Spec{
				Resources: map[string]value.ValueResource{
					"my-resource": {
						Identifier: value.ValueResourceIdentifier{
							Alias:        "my-resource",
							ResourceType: "bucket",
							Literal:      value.ValueString{Literal: "id"},
						},
					},
				},
			},
			expr: expr.Get{
				Name: "identifier",
				Object: expr.GetResource{
					Alias: "my-resource",
				},
			},
			out: value.ValueResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Literal:      value.ValueString{Literal: "id"},
			},
			children: []string{"my-resource"},
		},
		"resource, config": {
			build: build.Spec{
				Resources: map[string]value.ValueResource{
					"my-resource": {
						Config: value.ValueString{Literal: "config-val"},
					},
				},
			},
			expr: expr.Get{
				Name: "config",
				Object: expr.GetResource{
					Alias: "my-resource",
				},
			},
			out:      value.ValueString{Literal: "config-val"},
			children: []string{"my-resource"},
		},
		"resource, attrs, unresolved": {
			build: build.Spec{
				Resources: map[string]value.ValueResource{
					"my-resource": {
						Attrs: value.ValueUnresolved{
							Name: "attrs",
							Object: value.ValueResourceRef{
								Alias: "my-resource",
							},
						},
					},
				},
			},
			expr: expr.Get{
				Name: "attrs",
				Object: expr.GetResource{
					Alias: "my-resource",
				},
			},
			out: value.ValueUnresolved{
				Name: "attrs",
				Object: value.ValueResourceRef{
					Alias: "my-resource",
				},
			},
			children: []string{"my-resource"},
		},
		"resource, attrs": {
			build: build.Spec{
				Resources: map[string]value.ValueResource{
					"my-resource": {
						Attrs: value.ValueString{Literal: "foo"},
					},
				},
			},
			expr: expr.Get{
				Name: "attrs",
				Object: expr.GetResource{
					Alias: "my-resource",
				},
			},
			out:      value.ValueString{Literal: "foo"},
			children: []string{"my-resource"},
		},
		"unresolved": {
			build: build.Spec{
				Resources: map[string]value.ValueResource{
					"my-resource": {
						Attrs: value.ValueUnresolved{
							Name: "attrs",
							Object: value.ValueResourceRef{
								Alias: "my-resource",
							},
						},
					},
				},
			},
			expr: expr.Get{
				Name: "foo",
				Object: expr.Get{
					Name: "attrs",
					Object: expr.GetResource{
						Alias: "my-resource",
					},
				},
			},
			isError: true,
		},
		"unsupported type": {
			expr: expr.Get{
				Name:   "foo",
				Object: expr.String{Value: "foo"},
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
		build    build.Spec
		expr     expr.Type
		out      value.Value
		children []string
		isError  bool
	}{
		"unresolved": {
			build: build.Spec{
				Resources: map[string]value.ValueResource{
					"my-resource": {
						Attrs: value.ValueUnresolved{
							Name: "attrs",
							Object: value.ValueResourceRef{
								Alias: "my-resource",
							},
						},
					},
				},
			},
			expr: expr.IOGet{
				Name: "foo",
				Object: expr.Get{
					Name: "attrs",
					Object: expr.GetResource{
						Alias: "my-resource",
					},
				},
			},
			out: value.ValueUnresolved{
				Name: "foo",
				Object: value.ValueUnresolved{
					Name: "attrs",
					Object: value.ValueResourceRef{
						Alias: "my-resource",
					},
				},
			},
			children: []string{"my-resource"},
		},
		"map": {
			expr: expr.IOGet{
				Name: "foo",
				Object: expr.Map{
					Entries: map[string]expr.Type{
						"foo": expr.String{Value: "val"},
					},
				},
			},
			out:     value.ValueUnresolved{},
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
		build    build.Spec
		expr     expr.Type
		out      value.Value
		children []string
		isError  bool
	}{
		"valid": {
			build: build.Spec{},
			expr:  expr.String{Value: "hello world"},
			out:   value.ValueString{Literal: "hello world"},
		},
		// IOGet
		"io get: unresolved": {
			build: build.Spec{
				Resources: map[string]value.ValueResource{
					"my-resource": {
						Attrs: value.ValueUnresolved{
							Name: "attrs",
							Object: value.ValueResourceRef{
								Alias: "my-resource",
							},
						},
					},
				},
			},
			expr: expr.IOGet{
				Name: "foo",
				Object: expr.Get{
					Name: "attrs",
					Object: expr.GetResource{
						Alias: "my-resource",
					},
				},
			},
			out: value.ValueUnresolved{
				Name: "foo",
				Object: value.ValueUnresolved{
					Name: "attrs",
					Object: value.ValueResourceRef{
						Alias: "my-resource",
					},
				},
			},
			children: []string{"my-resource"},
		},
		"io get: map": {
			expr: expr.IOGet{
				Name: "foo",
				Object: expr.Map{
					Entries: map[string]expr.Type{
						"foo": expr.String{Value: "val"},
					},
				},
			},
			out:     value.ValueUnresolved{},
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
		build    build.Spec
		expr     expr.Type
		out      value.Value
		children []string
		isError  bool
	}{
		"provider identifier: valid": {
			expr: expr.ProviderIdentifier{
				Alias:   "my-provider",
				Name:    expr.String{Value: "gcp"},
				Version: expr.String{Value: "v0.0.1"},
			},
			out: value.ValueProviderIdentifier{
				Alias:   "my-provider",
				Name:    "gcp",
				Version: "v0.0.1",
			},
		},
		"provider identifier: get name": {
			expr: expr.ProviderIdentifier{
				Alias: "my-provider",
				Name: expr.Get{
					Name: "name",
					Object: expr.Map{
						Entries: map[string]expr.Type{
							"name": expr.String{Value: "gcp"},
						},
					},
				},
				Version: expr.String{Value: "v0.0.1"},
			},
			out: value.ValueProviderIdentifier{
				Alias:   "my-provider",
				Name:    "gcp",
				Version: "v0.0.1",
			},
		},
		"provider identifier: get version": {
			expr: expr.ProviderIdentifier{
				Alias: "my-provider",
				Name:  expr.String{Value: "gcp"},
				Version: expr.Get{
					Name: "version",
					Object: expr.Map{
						Entries: map[string]expr.Type{
							"version": expr.String{Value: "v0.0.1"},
						},
					},
				},
			},
			out: value.ValueProviderIdentifier{
				Alias:   "my-provider",
				Name:    "gcp",
				Version: "v0.0.1",
			},
		},
		"provider identifier: missing alias": {
			expr: expr.ProviderIdentifier{
				Alias:   "",
				Name:    expr.String{Value: "gcp"},
				Version: expr.String{Value: "v0.0.1"},
			},
			out:     value.ValueProviderIdentifier{},
			isError: true,
		},
		"provider identifier: name is not string": {
			expr: expr.ProviderIdentifier{
				Alias: "my-provider",
				Name: expr.Map{
					Entries: map[string]expr.Type{
						"name": expr.String{Value: "gcp"},
					},
				},
				Version: expr.String{Value: "v0.0.1"},
			},
			out:     value.ValueProviderIdentifier{},
			isError: true,
		},
		"provider identifier: version is not string": {
			expr: expr.ProviderIdentifier{
				Alias: "my-provider",
				Name:  expr.String{Value: "gcp"},
				Version: expr.Map{
					Entries: map[string]expr.Type{
						"name": expr.String{Value: "gcp"},
					},
				},
			},
			out:     value.ValueProviderIdentifier{},
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
		build    build.Spec
		expr     expr.Type
		out      value.Value
		children []string
		isError  bool
	}{
		"valid": {
			expr: expr.ResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Value:        expr.String{Value: "foo"},
			},
			out: value.ValueResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Literal:      value.ValueString{Literal: "foo"},
			},
			children: []string{"my-resource"},
		},
		"missing alias": {
			expr: expr.ResourceIdentifier{
				Alias:        "",
				ResourceType: "bucket",
				Value:        expr.String{Value: "foo"},
			},
			out:     value.ValueResourceIdentifier{},
			isError: true,
		},
		"missing resource type": {
			expr: expr.ResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "",
				Value:        expr.String{Value: "foo"},
			},
			out:     value.ValueResourceIdentifier{},
			isError: true,
		},
		"missing value": {
			expr: expr.ResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "",
				Value:        nil,
			},
			out:     value.ValueResourceIdentifier{},
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
		build    build.Spec
		expr     expr.Type
		out      value.Value
		children []string
		isError  bool
	}{
		"present": {
			build: build.Spec{
				Providers: map[string]value.ValueProvider{
					"my-provider": {
						Identifier: value.ValueProviderIdentifier{
							Alias:   "my-provider",
							Name:    "gcp",
							Version: "v0.0.1",
						},
					},
				},
			},
			expr: expr.GetProvider{
				Alias: "my-provider",
			},
			out: value.ValueProvider{
				Identifier: value.ValueProviderIdentifier{
					Alias:   "my-provider",
					Name:    "gcp",
					Version: "v0.0.1",
				},
			},
		},
		"not present": {
			build: build.Spec{
				Providers: map[string]value.ValueProvider{},
			},
			expr: expr.GetProvider{
				Alias: "my-provider",
			},
			out: value.ValueProvider{},
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
		build    build.Spec
		expr     expr.Type
		out      value.Value
		children []string
		isError  bool
	}{
		"present": {
			build: build.Spec{
				Resources: map[string]value.ValueResource{
					"my-resource": {
						Identifier: value.ValueResourceIdentifier{
							Alias:        "my-resource",
							ResourceType: "bucket",
							Literal:      value.ValueString{Literal: "foo"},
						},
					},
				},
			},
			expr: expr.GetResource{
				Alias: "my-resource",
			},
			out: value.ValueResource{
				Identifier: value.ValueResourceIdentifier{
					Alias:        "my-resource",
					ResourceType: "bucket",
					Literal:      value.ValueString{Literal: "foo"},
				},
			},
			children: []string{"my-resource"},
		},
		"not present": {
			expr: expr.GetResource{
				Alias: "my-resource",
			},
			out: value.ValueResource{},
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

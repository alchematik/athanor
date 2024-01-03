package interpreter_test

import (
	"context"
	"testing"

	"github.com/alchematik/athanor/blueprint/expr"
	"github.com/alchematik/athanor/build"
	"github.com/alchematik/athanor/build/value"
	"github.com/alchematik/athanor/interpreter"

	"github.com/stretchr/testify/require"
)

func TestInterpreter_Expr_Map(t *testing.T) {
	testCases := map[string]struct {
		build    build.Build
		expr     expr.Type
		out      value.Type
		children []string
		isError  bool
	}{
		"one entry": {
			build: build.Build{},
			expr: expr.Map{
				Entries: map[string]expr.Type{
					"foo": expr.String{Value: "bar"},
				},
			},
			out: value.Map{
				Entries: map[string]value.Type{
					"foo": value.String{Value: "bar"},
				},
			},
		},
		"several entries": {
			build: build.Build{},
			expr: expr.Map{
				Entries: map[string]expr.Type{
					"foo": expr.String{Value: "bar"},
					"bam": expr.String{Value: "baz"},
				},
			},
			out: value.Map{
				Entries: map[string]value.Type{
					"foo": value.String{Value: "bar"},
					"bam": value.String{Value: "baz"},
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
		build    build.Build
		expr     expr.Type
		out      value.Type
		children []string
		isError  bool
	}{
		"map, string value entry present": {
			build: build.Build{},
			expr: expr.Get{
				Name: "foo",
				Object: expr.Map{
					Entries: map[string]expr.Type{
						"foo": expr.String{Value: "bar"},
					},
				},
			},
			out: value.String{Value: "bar"},
		},
		"map, entry missing": {
			build: build.Build{},
			expr: expr.Get{
				Name: "foo",
				Object: expr.Map{
					Entries: map[string]expr.Type{},
				},
			},
			isError: true,
		},
		"resource, identifier": {
			build: build.Build{
				Resources: map[string]value.Resource{
					"my-resource": {
						Identifier: value.ResourceIdentifier{
							Alias:        "my-resource",
							ResourceType: "bucket",
							Value:        value.String{Value: "id"},
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
			out: value.ResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Value:        value.String{Value: "id"},
			},
			children: []string{"my-resource"},
		},
		"resource, config": {
			build: build.Build{
				Resources: map[string]value.Resource{
					"my-resource": {
						Config: value.String{Value: "config-val"},
					},
				},
			},
			expr: expr.Get{
				Name: "config",
				Object: expr.GetResource{
					Alias: "my-resource",
				},
			},
			out:      value.String{Value: "config-val"},
			children: []string{"my-resource"},
		},
		"resource, attrs, unresolved": {
			build: build.Build{
				Resources: map[string]value.Resource{
					"my-resource": {
						Attrs: value.Unresolved{
							Name: "attrs",
							Object: value.ResourceRef{
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
			out: value.Unresolved{
				Name: "attrs",
				Object: value.ResourceRef{
					Alias: "my-resource",
				},
			},
			children: []string{"my-resource"},
		},
		"resource, attrs": {
			build: build.Build{
				Resources: map[string]value.Resource{
					"my-resource": {
						Attrs: value.String{Value: "foo"},
					},
				},
			},
			expr: expr.Get{
				Name: "attrs",
				Object: expr.GetResource{
					Alias: "my-resource",
				},
			},
			out:      value.String{Value: "foo"},
			children: []string{"my-resource"},
		},
		"unresolved": {
			build: build.Build{
				Resources: map[string]value.Resource{
					"my-resource": {
						Attrs: value.Unresolved{
							Name: "attrs",
							Object: value.ResourceRef{
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
		build    build.Build
		expr     expr.Type
		out      value.Type
		children []string
		isError  bool
	}{
		"unresolved": {
			build: build.Build{
				Resources: map[string]value.Resource{
					"my-resource": {
						Attrs: value.Unresolved{
							Name: "attrs",
							Object: value.ResourceRef{
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
			out: value.Unresolved{
				Name: "foo",
				Object: value.Unresolved{
					Name: "attrs",
					Object: value.ResourceRef{
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
			out:     value.Unresolved{},
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
		build    build.Build
		expr     expr.Type
		out      value.Type
		children []string
		isError  bool
	}{
		"valid": {
			build: build.Build{},
			expr:  expr.String{Value: "hello world"},
			out:   value.String{Value: "hello world"},
		},
		// IOGet
		"io get: unresolved": {
			build: build.Build{
				Resources: map[string]value.Resource{
					"my-resource": {
						Attrs: value.Unresolved{
							Name: "attrs",
							Object: value.ResourceRef{
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
			out: value.Unresolved{
				Name: "foo",
				Object: value.Unresolved{
					Name: "attrs",
					Object: value.ResourceRef{
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
			out:     value.Unresolved{},
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
		build    build.Build
		expr     expr.Type
		out      value.Type
		children []string
		isError  bool
	}{
		"provider identifier: valid": {
			expr: expr.ProviderIdentifier{
				Alias:   "my-provider",
				Name:    expr.String{Value: "gcp"},
				Version: expr.String{Value: "v0.0.1"},
			},
			out: value.ProviderIdentifier{
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
			out: value.ProviderIdentifier{
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
			out: value.ProviderIdentifier{
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
			out:     value.ProviderIdentifier{},
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
			out:     value.ProviderIdentifier{},
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
			out:     value.ProviderIdentifier{},
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
		build    build.Build
		expr     expr.Type
		out      value.Type
		children []string
		isError  bool
	}{
		"valid": {
			expr: expr.ResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Value:        expr.String{Value: "foo"},
			},
			out: value.ResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "bucket",
				Value:        value.String{Value: "foo"},
			},
			children: []string{"my-resource"},
		},
		"missing alias": {
			expr: expr.ResourceIdentifier{
				Alias:        "",
				ResourceType: "bucket",
				Value:        expr.String{Value: "foo"},
			},
			out:     value.ResourceIdentifier{},
			isError: true,
		},
		"missing resource type": {
			expr: expr.ResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "",
				Value:        expr.String{Value: "foo"},
			},
			out:     value.ResourceIdentifier{},
			isError: true,
		},
		"missing value": {
			expr: expr.ResourceIdentifier{
				Alias:        "my-resource",
				ResourceType: "",
				Value:        nil,
			},
			out:     value.ResourceIdentifier{},
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
		build    build.Build
		expr     expr.Type
		out      value.Type
		children []string
		isError  bool
	}{
		"present": {
			build: build.Build{
				Providers: map[string]value.Provider{
					"my-provider": {
						Identifier: value.ProviderIdentifier{
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
			out: value.Provider{
				Identifier: value.ProviderIdentifier{
					Alias:   "my-provider",
					Name:    "gcp",
					Version: "v0.0.1",
				},
			},
		},
		"not present": {
			build: build.Build{
				Providers: map[string]value.Provider{},
			},
			expr: expr.GetProvider{
				Alias: "my-provider",
			},
			out: value.Provider{},
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
		build    build.Build
		expr     expr.Type
		out      value.Type
		children []string
		isError  bool
	}{
		"present": {
			build: build.Build{
				Resources: map[string]value.Resource{
					"my-resource": {
						Identifier: value.ResourceIdentifier{
							Alias:        "my-resource",
							ResourceType: "bucket",
							Value:        value.String{Value: "foo"},
						},
					},
				},
			},
			expr: expr.GetResource{
				Alias: "my-resource",
			},
			out: value.Resource{
				Identifier: value.ResourceIdentifier{
					Alias:        "my-resource",
					ResourceType: "bucket",
					Value:        value.String{Value: "foo"},
				},
			},
			children: []string{"my-resource"},
		},
		"not present": {
			expr: expr.GetResource{
				Alias: "my-resource",
			},
			out: value.Resource{},
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

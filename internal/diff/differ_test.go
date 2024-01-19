package diff_test

import (
	"testing"

	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/state"
	"github.com/stretchr/testify/require"
)

func TestResourceDiff(t *testing.T) {
	testCases := map[string]struct {
		from state.Type
		to   state.Type
		out  diff.Type
	}{
		"noop": {
			from: state.Resource{
				Config: state.String{Value: "config"},
				Exists: state.Bool{Value: true},
			},
			to: state.Resource{
				Config: state.String{Value: "config"},
				Exists: state.Bool{Value: true},
			},
			out: diff.Resource{
				From: state.Resource{
					Config: state.String{Value: "config"},
					Exists: state.Bool{Value: true},
				},
				To: state.Resource{
					Config: state.String{Value: "config"},
					Exists: state.Bool{Value: true},
				},
				ConfigDiff: diff.String{
					From:          state.String{Value: "config"},
					To:            state.String{Value: "config"},
					DiffOperation: diff.OperationNoop,
				},
				ExistsDiff: diff.Bool{
					From:          state.Bool{Value: true},
					To:            state.Bool{Value: true},
					DiffOperation: diff.OperationNoop,
				},
				DiffOperation: diff.OperationNoop,
			},
		},
		// "create":  {},
		"update": {
			from: state.Resource{
				Config: state.Map{
					Entries: map[string]state.Type{
						"foo": state.String{Value: "before"},
					},
				},
				Exists: state.Bool{Value: true},
			},
			to: state.Resource{
				Config: state.Map{
					Entries: map[string]state.Type{
						"foo": state.String{Value: "after"},
						"bar": state.String{Value: "baz"},
					},
				},
				Exists: state.Bool{Value: true},
			},
			out: diff.Resource{
				From: state.Resource{
					Config: state.Map{
						Entries: map[string]state.Type{
							"foo": state.String{Value: "before"},
						},
					},
					Exists: state.Bool{Value: true},
				},
				To: state.Resource{
					Config: state.Map{
						Entries: map[string]state.Type{
							"foo": state.String{Value: "after"},
							"bar": state.String{Value: "baz"},
						},
					},
					Exists: state.Bool{Value: true},
				},
				DiffOperation: diff.OperationUpdate,
				ExistsDiff: diff.Bool{
					From:          state.Bool{Value: true},
					To:            state.Bool{Value: true},
					DiffOperation: diff.OperationNoop,
				},
				ConfigDiff: diff.Map{
					From: state.Map{
						Entries: map[string]state.Type{
							"foo": state.String{Value: "before"},
						},
					},
					To: state.Map{
						Entries: map[string]state.Type{
							"foo": state.String{Value: "after"},
							"bar": state.String{Value: "baz"},
						},
					},
					DiffOperation: diff.OperationUpdate,
					Diffs: map[string]diff.Type{
						"foo": diff.String{
							From:          state.String{Value: "before"},
							To:            state.String{Value: "after"},
							DiffOperation: diff.OperationUpdate,
						},
						"bar": diff.String{
							From:          state.String{},
							To:            state.String{Value: "baz"},
							DiffOperation: diff.OperationCreate,
						},
					},
				},
			},
		},
		// "unknown": {},
		// "delete": {},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			out, err := diff.Diff(tc.from, tc.to)
			require.NoError(t, err)
			require.Equal(t, tc.out, out)
		})
	}
}

func TestStringDiff(t *testing.T) {
	testCases := []struct {
		description string
		from        state.Type
		to          state.Type
		out         diff.Type
	}{
		{
			description: "noop",
			from:        state.String{Value: "test"},
			to:          state.String{Value: "test"},
			out: diff.String{
				From:          state.String{Value: "test"},
				To:            state.String{Value: "test"},
				DiffOperation: diff.OperationNoop,
			},
		},
		{
			description: "create",
			from:        state.String{Value: ""},
			to:          state.String{Value: "test"},
			out: diff.String{
				From:          state.String{Value: ""},
				To:            state.String{Value: "test"},
				DiffOperation: diff.OperationCreate,
			},
		},
		{
			description: "delete",
			from:        state.String{Value: "test"},
			to:          state.String{Value: ""},
			out: diff.String{
				From:          state.String{Value: "test"},
				To:            state.String{Value: ""},
				DiffOperation: diff.OperationDelete,
			},
		},
		{
			description: "update",
			from:        state.String{Value: "test"},
			to:          state.String{Value: "test test"},
			out: diff.String{
				From:          state.String{Value: "test"},
				To:            state.String{Value: "test test"},
				DiffOperation: diff.OperationUpdate,
			},
		},
		{
			description: "unknown",
			from:        state.Unknown{},
			to:          state.String{Value: "foo"},
			out:         diff.Unknown{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			out, err := diff.Diff(tc.from, tc.to)
			require.NoError(t, err)
			require.Equal(t, tc.out, out)
		})
	}
}

func TestMap(t *testing.T) {
	one := state.String{Value: "one"}
	two := state.String{Value: "two"}
	four := state.String{Value: "four"}
	m1 := state.Map{
		Entries: map[string]state.Type{
			"one":   two,
			"three": four,
		},
	}
	empty := state.Map{
		Entries: map[string]state.Type{},
	}
	m2 := state.Map{
		Entries: map[string]state.Type{
			"one": one,
			"two": two,
		},
	}
	testCases := []struct {
		description string
		from        state.Type
		to          state.Type
		out         diff.Type
	}{
		{
			description: "noop",
			from:        m1,
			to:          m1,
			out: diff.Map{
				DiffOperation: diff.OperationNoop,
				From:          m1,
				To:            m1,
				Diffs: map[string]diff.Type{
					"one": diff.String{
						DiffOperation: diff.OperationNoop,
						From:          state.String{Value: "two"},
						To:            state.String{Value: "two"},
					},
					"three": diff.String{
						DiffOperation: diff.OperationNoop,
						From:          state.String{Value: "four"},
						To:            state.String{Value: "four"},
					},
				},
			},
		},
		{
			description: "create",
			from:        empty,
			to:          m1,
			out: diff.Map{
				DiffOperation: diff.OperationCreate,
				From:          empty,
				To:            m1,
				Diffs: map[string]diff.Type{
					"one": diff.String{
						DiffOperation: diff.OperationCreate,
						From:          state.String{},
						To:            state.String{Value: "two"},
					},
					"three": diff.String{
						DiffOperation: diff.OperationCreate,
						From:          state.String{},
						To:            state.String{Value: "four"},
					},
				},
			},
		},
		{
			description: "delete",
			from:        m1,
			to:          empty,
			out: diff.Map{
				DiffOperation: diff.OperationDelete,
				From:          m1,
				To:            empty,
				Diffs: map[string]diff.Type{
					"one": diff.String{
						DiffOperation: diff.OperationDelete,
						From:          state.String{Value: "two"},
						To:            state.String{},
					},
					"three": diff.String{
						DiffOperation: diff.OperationDelete,
						From:          state.String{Value: "four"},
						To:            state.String{},
					},
				},
			},
		},
		{
			description: "update",
			from:        m1,
			to:          m2,
			out: diff.Map{
				DiffOperation: diff.OperationUpdate,
				From:          m1,
				To:            m2,
				Diffs: map[string]diff.Type{
					"one": diff.String{
						DiffOperation: diff.OperationUpdate,
						From:          state.String{Value: "two"},
						To:            state.String{Value: "one"},
					},
					"three": diff.String{
						DiffOperation: diff.OperationDelete,
						From:          state.String{Value: "four"},
						To:            state.String{},
					},
					"two": diff.String{
						DiffOperation: diff.OperationCreate,
						From:          state.String{},
						To:            state.String{Value: "two"},
					},
				},
			},
		},
		{
			description: "has unknown",
			from: state.Map{
				Entries: map[string]state.Type{
					"unknown": state.Unknown{},
				},
			},
			to: state.Map{
				Entries: map[string]state.Type{
					"unknown": state.String{Value: "foo"},
				},
			},
			out: diff.Map{
				DiffOperation: diff.OperationUnknown,
				From: state.Map{
					Entries: map[string]state.Type{
						"unknown": state.Unknown{},
					},
				},
				To: state.Map{
					Entries: map[string]state.Type{
						"unknown": state.String{Value: "foo"},
					},
				},
				Diffs: map[string]diff.Type{
					"unknown": diff.Unknown{},
				},
			},
		},
		{
			description: "is unknown",
			from:        state.Unknown{},
			to: state.Map{
				Entries: map[string]state.Type{
					"foo": state.String{Value: "bar"},
				},
			},
			out: diff.Unknown{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			out, err := diff.Diff(tc.from, tc.to)
			require.NoError(t, err)
			require.Equal(t, tc.out, out)
		})
	}
}

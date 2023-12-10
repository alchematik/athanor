package diff_test

import (
	"testing"

	"github.com/alchematik/athanor/diff"
	"github.com/alchematik/athanor/state"
	"github.com/stretchr/testify/require"
)

func TestStringDiff(t *testing.T) {
	testCases := []struct {
		description string
		from        state.String
		to          state.String
		out         diff.Diff
	}{
		{
			description: "noop",
			from:        state.String{Value: "test"},
			to:          state.String{Value: "test"},
			out: diff.Diff{
				From:      state.String{Value: "test"},
				To:        state.String{Value: "test"},
				Operation: diff.OperationNoop,
			},
		},
		{
			description: "create",
			from:        state.String{Value: ""},
			to:          state.String{Value: "test"},
			out: diff.Diff{
				From:      state.String{Value: ""},
				To:        state.String{Value: "test"},
				Operation: diff.OperationCreate,
			},
		},
		{
			description: "delete",
			from:        state.String{Value: "test"},
			to:          state.String{Value: ""},
			out: diff.Diff{
				From:      state.String{Value: "test"},
				To:        state.String{Value: ""},
				Operation: diff.OperationDelete,
			},
		},
		{
			description: "update",
			from:        state.String{Value: "test"},
			to:          state.String{Value: "test test"},
			out: diff.Diff{
				From:      state.String{Value: "test"},
				To:        state.String{Value: "test test"},
				Operation: diff.OperationUpdate,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			out, err := diff.String(tc.from, tc.to)
			require.NoError(t, err)
			require.Equal(t, tc.out, out)
		})
	}
}

func TestMap(t *testing.T) {
	one := state.String{Value: "one"}
	two := state.String{Value: "two"}
	// three := state.String{Value: "three"}
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
		from        state.Map
		to          state.Map
		out         diff.Diff
	}{
		{
			description: "noop",
			from:        m1,
			to:          m1,
			out: diff.Diff{
				Operation: diff.OperationNoop,
				From:      m1,
				To:        m1,
				Diffs: []diff.Diff{
					{
						Name:      "one",
						Operation: diff.OperationNoop,
						From:      state.String{Value: "two"},
						To:        state.String{Value: "two"},
					},
					{
						Name:      "three",
						Operation: diff.OperationNoop,
						From:      state.String{Value: "four"},
						To:        state.String{Value: "four"},
					},
				},
			},
		},
		{
			description: "create",
			from:        empty,
			to:          m1,
			out: diff.Diff{
				Operation: diff.OperationCreate,
				From:      empty,
				To:        m1,
				Diffs: []diff.Diff{
					{
						Name:      "one",
						Operation: diff.OperationCreate,
						From:      state.Nil{},
						To:        state.String{Value: "two"},
					},
					{
						Name:      "three",
						Operation: diff.OperationCreate,
						From:      state.Nil{},
						To:        state.String{Value: "four"},
					},
				},
			},
		},
		{
			description: "delete",
			from:        m1,
			to:          empty,
			out: diff.Diff{
				Operation: diff.OperationDelete,
				From:      m1,
				To:        empty,
				Diffs: []diff.Diff{
					{
						Name:      "one",
						Operation: diff.OperationDelete,
						From:      state.String{Value: "two"},
						To:        state.Nil{},
					},
					{
						Name:      "three",
						Operation: diff.OperationDelete,
						From:      state.String{Value: "four"},
						To:        state.Nil{},
					},
				},
			},
		},
		{
			description: "update",
			from:        m1,
			to:          m2,
			out: diff.Diff{
				Operation: diff.OperationUpdate,
				From:      m1,
				To:        m2,
				Diffs: []diff.Diff{
					{
						Name:      "one",
						Operation: diff.OperationUpdate,
						From:      state.String{Value: "two"},
						To:        state.String{Value: "one"},
					},
					{
						Name:      "three",
						Operation: diff.OperationDelete,
						From:      state.String{Value: "four"},
						To:        state.Nil{},
					},
					{
						Name:      "two",
						Operation: diff.OperationCreate,
						From:      state.Nil{},
						To:        state.String{Value: "two"},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			out, err := diff.Map(tc.from, tc.to)
			require.NoError(t, err)
			require.Equal(t, tc.out, out)
		})
	}
}

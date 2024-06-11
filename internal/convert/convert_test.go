package convert_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	external "github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/convert"
)

func TestConvertBoolExpr_Bool(t *testing.T) {
	testCases := []struct {
		name string
		in   external.Expr
		out  ast.Expr[bool]
	}{
		{
			name: "bool literal - true",
			in: external.Expr{
				Value: external.BoolLiteral{
					Value: true,
				},
			},
			out: ast.Literal[bool]{
				Value: true,
			},
		},
		{
			name: "bool literal - false",
			in: external.Expr{
				Value: external.BoolLiteral{
					Value: false,
				},
			},
			out: ast.Literal[bool]{
				Value: false,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			out, err := convert.ConvertBoolExpr("", test.in)
			require.NoError(t, err)
			require.Equal(t, test.out, out)
		})
	}
}

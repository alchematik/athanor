package ast_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alchematik/athanor/ast"
)

func TestExpr_Unmarshal(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		expected ast.Expr
	}{
		{
			name: "string",
			in: `{
			  "type": "string", "value":{"string_literal": "hi"}
			}`,
			expected: ast.Expr{
				Type: "string",
				Value: ast.StringLiteral{
					Value: "hi",
				},
			},
		},
		{
			name: "map",
			in: `{
		    "type": "map",
		    "value": {
		      "map_collection": {
		        "foo": {
		          "type": "string",
		          "value": {
		            "string_literal": "bar"
		          }
		        }
		      }
		    }
		  }`,
			expected: ast.Expr{
				Type: "map",
				Value: ast.MapCollection{
					Value: map[string]ast.Expr{
						"foo": {
							Type: "string",
							Value: ast.StringLiteral{
								Value: "bar",
							},
						},
					},
				},
			},
		},
		{
			name: "resource",
			in: `{
		    "type": "resource",
		    "value": {
		      "identifier": {
		        "type": "map",
		        "value": {
		          "map_collection": {
		            "foo": {
		              "type": "string",
		              "value": {
		                "string_literal": "bar"
		              }
		            },
		            "baz": {
		              "type": "integer",
		              "value": {
		                "integer_literal": 10
		              }
		            }
		          }
		        }
		      },
		      "config": {
		        "type": "map",
		        "value": {
		          "map_collection": {
		            "enable": {
		              "type": "bool",
		              "value": {
		                "bool_literal": true
		              }
		            }
		          }
		        }
		      }
		    }
		  }`,
			expected: ast.Expr{
				Type: "resource",
				Value: ast.Resource{
					Identifier: ast.Expr{
						Type: "map",
						Value: ast.MapCollection{
							Value: map[string]ast.Expr{
								"foo": {
									Type: "string",
									Value: ast.StringLiteral{
										Value: "bar",
									},
								},
								"baz": {
									Type: "integer",
									Value: ast.IntegerLiteral{
										Value: 10,
									},
								},
							},
						},
					},
					Config: ast.Expr{
						Type: "map",
						Value: ast.MapCollection{
							Value: map[string]ast.Expr{
								"enable": {
									Type: "bool",
									Value: ast.BoolLiteral{
										Value: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var expr ast.Expr
			err := json.Unmarshal([]byte(test.in), &expr)
			require.NoError(t, err)
			require.Equal(t, test.expected, expr)
		})
	}
}

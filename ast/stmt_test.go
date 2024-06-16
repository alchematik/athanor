package ast_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alchematik/athanor/ast"
)

func TestStmt_Unmarshal(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		expected ast.Stmt
	}{
		{
			name: "resource",
			in: `{
			  "type": "resource",
			  "value": {
			    "name": "my-resource",
			    "resource": {
			      "type": "resource",
			      "value": {
			    		"exists": {
			      		"type": "bool",
			      		"value": {
			        		"bool_literal": true
			      		}
			    		},
			        "identifier": {
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
			        },
			        "config": {
			          "type": "map",
			          "value": {
			            "map_collection": {
			              "enabled": {
			                "type": "bool",
			                "value": {
			                  "bool_literal": true
			                }
			              }
			            }
			          }
			        }
			      }
			    }
			  } 
	    }`,
			expected: ast.Stmt{
				Type: "resource",
				Value: ast.DeclareResource{
					Name: "my-resource",
					Resource: ast.Expr{
						Type: "resource",
						Value: ast.Resource{
							Identifier: ast.Expr{
								Type: "map",
								Value: ast.MapCollection{
									Value: map[string]ast.Expr{
										"foo": {
											Type:  "string",
											Value: ast.StringLiteral{Value: "bar"},
										},
									},
								},
							},
							Config: ast.Expr{
								Type: "map",
								Value: ast.MapCollection{
									Value: map[string]ast.Expr{
										"enabled": {
											Type:  "bool",
											Value: ast.BoolLiteral{Value: true},
										},
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
			var stmt ast.Stmt
			err := json.Unmarshal([]byte(test.in), &stmt)
			require.NoError(t, err)
			require.Equal(t, test.expected, stmt)
		})
	}
}

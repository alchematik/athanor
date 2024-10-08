package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/alchematik/athanor/ast"
)

func main() {
	f, err := os.Create("blueprint.json")
	if err != nil {
		log.Fatalf("error creating file: %s", err)
	}

	bp := ast.Blueprint{
		Stmts: []ast.Stmt{
			{
				Type: "resource",
				Value: ast.DeclareResource{
					Name: "my-sub-resource",
					Type: ast.Expr{
						Type: "string",
						Value: ast.StringLiteral{
							Value: "bucket",
						},
					},
					Exists: ast.Expr{
						Type: "bool",
						Value: ast.BoolLiteral{
							Value: true,
						},
					},
					Provider: ast.Expr{
						Type: "provider",
						Value: ast.Provider{
							Name: ast.Expr{
								Type: "string",
								Value: ast.StringLiteral{
									Value: "google-cloud",
								},
							},
							Version: ast.Expr{
								Type: "string",
								Value: ast.StringLiteral{
									Value: "v0.0.1",
								},
							},
						},
					},
					Identifier: ast.Expr{
						Type: "map",
						Value: ast.MapCollection{
							Value: map[string]ast.Expr{
								"name": {
									Type: "string",
									Value: ast.StringLiteral{
										Value: "my-resource-name",
									},
								},
							},
						},
					},
					Config: ast.Expr{
						Type: "map",
						Value: ast.MapCollection{
							Value: map[string]ast.Expr{
								"thing": {
									Type: "string",
									Value: ast.StringLiteral{
										Value: "my-config",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := json.NewEncoder(f).Encode(bp); err != nil {
		log.Fatalf("error writing to file: %v", err)
	}
}

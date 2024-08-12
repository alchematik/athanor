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
				Type: "build",
				Value: ast.DeclareBuild{
					Name: "sub-build",
					Exists: ast.Expr{
						Type: "bool",
						Value: ast.BoolLiteral{
							Value: true,
						},
					},
					Runtimeinput: ast.Expr{
						Type: "map",
						Value: ast.MapCollection{
							Value: map[string]ast.Expr{},
						},
					},
					BlueprintSource: ast.BlueprintSource{
						LocalFile: ast.BlueprintSourceLocalFile{
							Path: "./example/gcp/sub/main.wasm",
						},
					},
				},
			},
			{
				Type: "resource",
				Value: ast.DeclareResource{
					Name: "my-resource",
					Exists: ast.Expr{
						Type: "bool",
						Value: ast.BoolLiteral{
							Value: true,
						},
					},
					Resource: ast.Expr{
						Type: "resource",
						Value: ast.Resource{
							Type: ast.Expr{
								Type: "string",
								Value: ast.StringLiteral{
									Value: "bucket",
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
										"region": {
											Type: "string",
											Value: ast.StringLiteral{
												Value: "us-west-2",
											},
										},
										"project": {
											Type: "string",
											Value: ast.StringLiteral{
												Value: "1234",
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
										"test": {
											Type: "string",
											Value: ast.StringLiteral{
												Value: "hey",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			{
				Type: "resource",
				Value: ast.DeclareResource{
					Name: "my-other-resource",
					Exists: ast.Expr{
						Type: "bool",
						Value: ast.BoolLiteral{
							Value: true,
						},
					},
					Resource: ast.Expr{
						Type: "resource",
						Value: ast.Resource{
							Type: ast.Expr{
								Type: "string",
								Value: ast.StringLiteral{
									Value: "bucket",
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
												Value: "my-other-resource-name",
											},
										},
										"region": {
											Type: "string",
											Value: ast.StringLiteral{
												Value: "us-east-1",
											},
										},
										"project": {
											Type: "string",
											Value: ast.StringLiteral{
												Value: "1234",
											},
										},
									},
								},
							},
							Config: ast.Expr{
								Type: "map",
								Value: ast.MapCollection{
									Value: map[string]ast.Expr{
										"other-thing": {
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
			},
		},
	}

	if err := json.NewEncoder(f).Encode(bp); err != nil {
		log.Fatalf("error writing to file: %v", err)
	}
}

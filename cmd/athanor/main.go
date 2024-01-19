package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/alchematik/athanor/diff"
	"github.com/alchematik/athanor/evaluator"
	api "github.com/alchematik/athanor/internal/api/resource"
	"github.com/alchematik/athanor/internal/ast"
	consumerpb "github.com/alchematik/athanor/internal/gen/go/proto/blueprint/v1"
	translatorpb "github.com/alchematik/athanor/internal/gen/go/proto/translator/v1"
	"github.com/alchematik/athanor/interpreter"
	plug "github.com/alchematik/athanor/plugin"
	"github.com/alchematik/athanor/reconcile"

	"github.com/urfave/cli/v2"
)

func convertBlueprint(bp *consumerpb.Blueprint) (ast.Blueprint, error) {
	out := ast.Blueprint{}
	for _, stmt := range bp.GetStmts() {
		converted, err := convertStmt(stmt)
		if err != nil {
			return ast.Blueprint{}, err
		}

		out.Stmts = append(out.Stmts, converted)
	}

	return out, nil
}

func convertStmt(st *consumerpb.Stmt) (ast.Stmt, error) {
	switch s := st.GetType().(type) {
	case *consumerpb.Stmt_Resource:
		ex, err := convertExpr(s.Resource.GetExpr())
		if err != nil {
			return nil, err
		}

		return ast.StmtResource{
			Expr: ex,
		}, nil
	case *consumerpb.Stmt_Build:
		ex, err := convertExpr(s.Build.GetBlueprint())
		if err != nil {
			return nil, err
		}

		inputs := map[string]ast.Expr{}
		for name, inputExpr := range s.Build.GetInputs() {
			input, err := convertExpr(inputExpr)
			if err != nil {
				return nil, err
			}

			inputs[name] = input
		}

		return ast.StmtBuild{
			Alias:     s.Build.GetAlias(),
			Blueprint: ex,
			Inputs:    inputs,
		}, nil
	default:
		return nil, fmt.Errorf("invalid stmt: %T", st.GetType())
	}
}

func convertExpr(ex *consumerpb.Expr) (ast.Expr, error) {
	switch e := ex.GetType().(type) {
	case *consumerpb.Expr_Blueprint:
		stmts := make([]ast.Stmt, len(e.Blueprint.GetStmts()))
		for i, s := range e.Blueprint.GetStmts() {
			converted, err := convertStmt(s)
			if err != nil {
				return nil, err
			}

			stmts[i] = converted
		}

		return ast.ExprBlueprint{Stmts: stmts}, nil
	case *consumerpb.Expr_Provider:
		return ast.ExprProvider{
			Name:    e.Provider.GetName(),
			Version: e.Provider.GetVersion(),
		}, nil
	case *consumerpb.Expr_Resource:
		provider, err := convertExpr(e.Resource.GetProvider())
		if err != nil {
			return nil, err
		}

		id, err := convertExpr(e.Resource.GetIdentifier())
		if err != nil {
			return nil, err
		}

		config, err := convertExpr(e.Resource.GetConfig())
		if err != nil {
			return nil, err
		}

		exists, err := convertExpr(e.Resource.GetExists())
		if err != nil {
			return nil, err
		}

		return ast.ExprResource{
			Provider:   provider,
			Identifier: id,
			Config:     config,
			Exists:     exists,
		}, nil
	case *consumerpb.Expr_ResourceIdentifier:
		val, err := convertExpr(e.ResourceIdentifier.GetValue())
		if err != nil {
			return ast.ExprResourceIdentifier{}, err
		}

		return ast.ExprResourceIdentifier{
			Alias:        e.ResourceIdentifier.GetAlias(),
			ResourceType: e.ResourceIdentifier.GetType(),
			Value:        val,
		}, nil
	case *consumerpb.Expr_StringLiteral:
		return ast.ExprString{Value: e.StringLiteral}, nil
	case *consumerpb.Expr_BoolLiteral:
		return ast.ExprBool{Value: e.BoolLiteral}, nil
	case *consumerpb.Expr_File:
		return ast.ExprFile{Path: e.File.Path}, nil
	case *consumerpb.Expr_Map:
		m := ast.ExprMap{Entries: map[string]ast.Expr{}}
		for k, v := range e.Map.GetEntries() {
			var err error
			m.Entries[k], err = convertExpr(v)
			if err != nil {
				return nil, err
			}
		}

		return m, nil
	case *consumerpb.Expr_Get:
		obj, err := convertExpr(e.Get.GetObject())
		if err != nil {
			return nil, err
		}

		g := ast.ExprGet{
			Name:   e.Get.GetName(),
			Object: obj,
		}

		return g, nil
	case *consumerpb.Expr_Nil:
		return ast.ExprNil{}, nil
	default:
		return nil, fmt.Errorf("invalid expr: %T", ex.GetType())
	}
}

func main() {
	app := cli.App{
		Name: "athanor",
		Commands: []*cli.Command{
			{
				Name: "provider",
				Subcommands: []*cli.Command{
					{
						Name: "generate",
						Action: func(ctx *cli.Context) error {
							p := ctx.Args().First()
							f, err := os.ReadFile(p)
							if err != nil {
								return err
							}

							type ClientSDK struct {
								OutputPath string `json:"output_path"`
								Translator struct {
									Name    string `json:"name"`
									Version string `json:"version"`
								} `json:"translator"`
							}

							type Config struct {
								InputPath  string `json:"input_path"`
								OutputPath string `json:"output_path"`
								Translator struct {
									Name    string `json:"name"`
									Version string `json:"version"`
								} `json:"translator"`
								Args           map[string]string `json:"args"`
								ClientSDK      []ClientSDK       `json:"client_sdk"`
								TranslatorsDir string            `json:"translators_dir"`
							}

							var c Config
							if err := json.Unmarshal(f, &c); err != nil {
								return err
							}

							translatorPlugManager := plug.Translator{
								Dir: c.TranslatorsDir,
							}

							client, err := translatorPlugManager.Client(c.Translator.Name, c.Translator.Version)
							if err != nil {
								return err
							}

							tempFile, err := os.CreateTemp("", "")
							if err != nil {
								return err
							}

							defer os.Remove(tempFile.Name())

							_, err = client.TranslateProviderSchema(ctx.Context, &translatorpb.TranslateProviderSchemaRequest{
								OutputPath: tempFile.Name(),
								InputPath:  c.InputPath,
							})
							if err != nil {
								return err
							}

							_, err = client.GenerateProviderSDK(ctx.Context, &translatorpb.GenerateProviderSDKRequest{
								InputPath:  tempFile.Name(),
								OutputPath: c.OutputPath,
								Args:       c.Args,
							})
							if err != nil {
								return err
							}

							for _, clientSDK := range c.ClientSDK {
								trans, err := translatorPlugManager.Client(clientSDK.Translator.Name, clientSDK.Translator.Version)
								if err != nil {
									return err
								}

								_, err = trans.GenerateConsumerSDK(ctx.Context, &translatorpb.GenerateConsumerSDKRequest{
									InputPath:  tempFile.Name(),
									OutputPath: clientSDK.OutputPath,
								})
								if err != nil {
									return err
								}
							}

							return nil
						},
					},
				},
			},
			{
				Name: "blueprint",
				Subcommands: []*cli.Command{
					{
						Name: "reconcile",
						Action: func(ctx *cli.Context) error {
							p := ctx.Args().First()
							f, err := os.ReadFile(p)
							if err != nil {
								return err
							}

							type Config struct {
								InputPath  string `json:"input_path"`
								Translator struct {
									Name    string `json:"name"`
									Version string `json:"version"`
								} `json:"translator"`
								TranslatorsDir string `json:"translators_dir"`
								ProvidersDir   string `json:"providers_dir"`
							}

							var c Config
							if err := json.Unmarshal(f, &c); err != nil {
								return err
							}

							translatorPlugManager := plug.Translator{
								Dir: c.TranslatorsDir,
							}

							client, err := translatorPlugManager.Client(c.Translator.Name, c.Translator.Version)
							if err != nil {
								return err
							}

							tempFile, err := os.CreateTemp("", "")
							if err != nil {
								return err
							}

							// defer os.Remove(tempFile.Name())

							fmt.Printf("TEMP FILE >>>>>>>>>>>>>>> %v\n", tempFile.Name())

							_, err = client.TranslateBlueprint(ctx.Context, &translatorpb.TranslateBlueprintRequest{
								InputPath:  c.InputPath,
								OutputPath: tempFile.Name(),
							})
							if err != nil {
								return err
							}

							blueprintData, err := os.ReadFile(tempFile.Name())
							if err != nil {
								return err
							}

							var blueprint consumerpb.Blueprint
							if err := json.Unmarshal(blueprintData, &blueprint); err != nil {
								return err
							}

							data, err := json.MarshalIndent(&blueprint, "", "  ")
							if err != nil {
								return err
							}

							// fmt.Printf("IN >>>>>>>>>>>> %v\n", string(data))

							bp, err := convertBlueprint(&blueprint)
							if err != nil {
								return err
							}

							in := interpreter.Interpreter{}
							build, err := in.Interpret(ctx.Context, bp)
							if err != nil {
								return err
							}

							fmt.Printf("dep map >>> %+v\n", build.DependencyMap)

							eval := evaluator.Evaluator{
								ResourceAPI: &api.Unresolved{},
							}

							desiredEnv, err := eval.Evaluate(ctx.Context, build)
							if err != nil {
								return err
							}

							data, err = json.MarshalIndent(desiredEnv, "", "  ")
							if err != nil {
								return err
							}

							fmt.Printf("desired state >>>>>>>>>>>> %v\n", string(data))

							remoteEval := evaluator.Evaluator{
								ResourceAPI: api.API{
									ProviderPluginManager: plug.Provider{
										Dir: c.ProvidersDir,
									},
								},
								// ResourceEvaluator: evaluator.RemoteResourceEvaluator{
								// 	ValueResolver:     evaluator.ValueResolver{},
								// 	ProviderPluginDir: ".backends",
								// },
							}

							remoteEnv, err := remoteEval.Evaluate(ctx.Context, build)
							if err != nil {
								return err
							}

							data, err = json.MarshalIndent(remoteEnv, "", "  ")
							if err != nil {
								return err
							}

							// fmt.Printf("actual state <<<<<<<<<<< %v\n", string(data))

							// TODO: create diff between local state and remote state.

							d, err := diff.Diff(remoteEnv, desiredEnv)
							if err != nil {
								return err
							}

							data, err = json.MarshalIndent(d, "", "  ")
							if err != nil {
								return err
							}

							fmt.Printf("DIFF >>>>>>>>>> %v\n", string(data))

							reconciler := reconcile.Reconciler{
								ResourceAPI: api.API{
									ProviderPluginManager: plug.Provider{
										Dir: c.ProvidersDir,
									},
								},
							}
							reconciledState, err := reconciler.ReconcileEnvironment(ctx.Context, d.(diff.Environment))
							if err != nil {
								return err
							}

							data, err = json.MarshalIndent(reconciledState, "", "  ")
							if err != nil {
								return err
							}

							fmt.Printf("RECONCILED >>>>>>>>>> %v\n", string(data))

							return nil
						},
					},
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

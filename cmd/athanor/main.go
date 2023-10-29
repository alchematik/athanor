package main

import (
	// "encoding/gob"
	"errors"
	"fmt"
	"golang.org/x/sync/errgroup"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-hclog"
	// "sort"

	"github.com/alchematik/athanor/internal/parser"
	"github.com/alchematik/athanor/internal/provider"
	"github.com/alchematik/athanor/provider"

	"github.com/dominikbraun/graph"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/urfave/cli/v2"
	"github.com/zclconf/go-cty/cty"
	// "golang.org/x/mod/semver"
)

var blockTypes = []hcl.BlockHeaderSchema{
	{
		Type:       "provider",
		LabelNames: []string{"name"},
	},
	{
		Type:       "id",
		LabelNames: []string{"provider", "resource", "name"},
	},
	{
		Type:       "create",
		LabelNames: []string{"provider", "resource"},
	},
	{
		Type:       "update",
		LabelNames: []string{"provider", "resource"},
	},
	{
		Type:       "delete",
		LabelNames: []string{"provider", "resource"},
	},
}

func main() {
	app := cli.App{
		Name: "athanor",
		Commands: []*cli.Command{
			{
				Name: "blueprint",
				Subcommands: []*cli.Command{
					{
						Name: "show",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name: "providers",
							},
						},
						Action: func(ctx *cli.Context) error {
							dir := ctx.Args().First()
							if dir == "" {
								dir = "."
							}
							var files []parser.File
							err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
								if info.IsDir() {
									return nil
								}
								if err != nil {
									return err
								}

								ext := filepath.Ext(path)
								if ext != ".hcl" {
									return nil
								}

								data, err := os.ReadFile(path)
								if err != nil {
									return err
								}

								files = append(files, parser.File{
									Path:    path,
									Content: data,
								})

								return nil
							})
							if err != nil {
								return err
							}

							if len(files) == 0 {
								return errors.New("no files found")
							}

							providersPath := ctx.String("providers")
							if providersPath == "" {
								return errors.New("must provide path to providers")
							}

							resourcesSchema := &hcl.BodySchema{Blocks: blockTypes}
							p := hclparse.NewParser()
							var blocks []*hcl.Block
							for _, f := range files {
								parsed, diag := p.ParseHCL(f.Content, f.Path)
								if diag.HasErrors() {
									return diag
								}

								content, diag := parsed.Body.Content(resourcesSchema)
								if diag.HasErrors() {
									return diag
								}

								blocks = append(blocks, content.Blocks...)
							}

							idBlocks := map[string]map[string]*hcl.Block{}
							type Provider struct {
								Alias   string `hcl:"name"`
								Version string `hcl:"version"`
							}
							providers := map[string]map[string]Provider{}

							opBlocks := map[string][]*hcl.Block{}

							for _, b := range blocks {
								switch b.Type {
								case "id":
									provider := b.Labels[0]
									resource := b.Labels[1]
									name := b.Labels[2]
									resourceKey := provider + "." + resource
									resourceMap, ok := idBlocks[resourceKey]
									if !ok {
										resourceMap = map[string]*hcl.Block{}
										idBlocks[resourceKey] = resourceMap
									}

									_, ok = resourceMap[name]
									if ok {
										return fmt.Errorf("dupe: %v.%v", resourceKey, name)
									}

									idBlocks[resourceKey][name] = b
								case "provider":
									providerName := b.Labels[0]
									var p Provider
									if diag := gohcl.DecodeBody(b.Body, nil, &p); diag.HasErrors() {
										return diag
									}
									m, ok := providers[providerName]
									if !ok {
										m = map[string]Provider{}
										providers[providerName] = m
									}
									m[p.Alias] = p
								case "create":
									provider := b.Labels[0]
									opBlocks[provider] = append(opBlocks[provider], b)
								}
							}

							// Load plugins based on provider.

							evalCtx := &hcl.EvalContext{
								Variables: map[string]cty.Value{},
							}

							pluginClients := map[string]*plugin.Client{}
							providerClients := map[string]*provider.ProviderRPCClient{}

							for providerType, providerMap := range providers {
								for alias, providerData := range providerMap {
									fp := filepath.Join(providersPath, providerType, providerData.Version, "provider")
									var handshakeConfig = plugin.HandshakeConfig{
										ProtocolVersion:  1,
										MagicCookieKey:   "BASIC_PLUGIN",
										MagicCookieValue: "hello",
									}
									client := plugin.NewClient(&plugin.ClientConfig{
										HandshakeConfig: handshakeConfig,
										Plugins: map[string]plugin.Plugin{
											"provider": &provider.ProviderPlugin{},
										},
										Cmd: exec.Command(fp),
										// Logger: hclog.New(&hclog.LoggerOptions{
										// 	Output: os.Stdout,
										// 	Level:  hclog.Trace,
										// }),
										Logger: hclog.NewNullLogger(),
									})

									pluginClients[alias] = client

									rpcClient, err := client.Client()
									if err != nil {
										return err
									}

									raw, err := rpcClient.Dispense("provider")
									if err != nil {
										return err
									}

									providerClient, ok := raw.(*provider.ProviderRPCClient)
									if !ok {
										return fmt.Errorf("not a client: %T", raw)
									}

									providerClients[alias] = providerClient
								}
							}

							// var ids []any
							for _, providerMap := range providers {
								for alias := range providerMap {
									schema, err := providerClients[alias].Schema()
									if err != nil {
										return err
									}

									dag := graph.New(graph.StringHash, graph.Directed(), graph.Acyclic(), graph.PreventCycles())
									if err := dag.AddVertex("root"); err != nil {
										return err
									}
									for name := range schema.Resources {
										if err := dag.AddVertex(name); err != nil {
											return err
										}
									}

									for name, r := range schema.Resources {
										if len(r.DependsOn) == 0 {
											if err := dag.AddEdge("root", name); err != nil {
												return err
											}
											continue
										}

										for _, dep := range r.DependsOn {
											if err := dag.AddEdge(dep, name); err != nil {
												return err
											}
										}
									}

									resourceNames, err := graph.TopologicalSort(dag)
									if err != nil {
										return err
									}

									resourceNames = resourceNames[1:]

									for _, rn := range resourceNames {
										blocks := idBlocks[alias+"."+rn]
										s := schema.Resources[rn].IdentifierFields
										var hclAttrs []hcl.AttributeSchema
										for _, f := range s {
											hclAttrs = append(hclAttrs, hcl.AttributeSchema{Name: f.Name})
										}
										for _, b := range blocks {
											content, diag := b.Body.Content(&hcl.BodySchema{Attributes: hclAttrs})
											if diag.HasErrors() {
												return diag
											}

											var fvs []provider.FieldValue
											for _, f := range s {
												if attr, ok := content.Attributes[f.Name]; ok {
													fv, err := provider.DecodeField(evalCtx, attr.Expr, f, schema)
													if err != nil {
														return err
													}
													fvs = append(fvs, fv)
												}
											}

											val, err := provider.FieldValuesToCtyValue(fvs)
											if err != nil {
												return err
											}

											provider.AddIdentifierValueToEvalCtx(evalCtx, b, val)
										}
									}
								}
							}

							var ops []provider.Operation
							for _, p := range providers {
								for alias := range p {
									providerClient := providerClients[alias]
									for _, b := range opBlocks[alias] {
										bs := &hcl.BodySchema{
											Attributes: []hcl.AttributeSchema{
												{Name: "id"},
												{Name: "version"},
											},
											Blocks: []hcl.BlockHeaderSchema{
												{
													Type: "config",
												},
											},
										}
										content, diag := b.Body.Content(bs)
										if diag.HasErrors() {
											return diag
										}

										schema, err := providerClient.Schema()
										if err != nil {
											return err
										}

										idAttr := content.Attributes["id"]
										t := b.Labels[1]
										ivf, err := provider.DecodeField(evalCtx, idAttr.Expr, provider.Field{Name: "id", Type: "identifier"}, schema)
										if err != nil {
											return err
										}

										versionAttr := content.Attributes["version"]
										vfv, err := provider.DecodeField(evalCtx, versionAttr.Expr, provider.Field{Name: "version", Type: "string"}, schema)
										if err != nil {
											return err
										}

										var cfv []provider.FieldValue
										for _, b := range content.Blocks {
											if b.Type == "config" {
												var attrs []hcl.AttributeSchema
												for _, f := range schema.Resources[t].ConfigFields {
													attrs = append(attrs, hcl.AttributeSchema{Name: f.Name})
												}

												configContent, diag := b.Body.Content(&hcl.BodySchema{Attributes: attrs})
												if diag.HasErrors() {
													return diag
												}

												for _, f := range schema.Resources[t].ConfigFields {
													if attr, ok := configContent.Attributes[f.Name]; ok {
														fv, err := provider.DecodeField(evalCtx, attr.Expr, f, schema)
														if err != nil {
															return err
														}
														cfv = append(cfv, fv)
													}
												}

												fmt.Printf("config: %+v\n", cfv)
											}
										}
										op := provider.Operation{
											Provider:         alias,
											ResourceType:     b.Labels[1],
											IdentifierFields: ivf.Value.([]provider.FieldValue),
											ConfigFields:     cfv,
											Version:          vfv.Value.(string),
											Action:           b.Type,
										}

										ops = append(ops, op)
									}
								}
							}

							state := provider.State{
								Resources: map[string]provider.Resource{},
							}
							for _, op := range ops {
								state.Apply(op)
							}

							nextState := provider.State{
								Resources: map[string]provider.Resource{},
							}
							wg := errgroup.Group{}
							lock := sync.Mutex{}
							for id, v := range state.Resources {
								id := id
								v := v
								wg.Go(func() error {
									return backoff.Retry(func() error {

										lock.Lock()
										client := providerClients[v.Provider]
										lock.Unlock()

										res, err := client.GetResource(provider.GetResurceInput{
											IdentifierFields: v.IdentifierFields,
											ResourceType:     v.Type,
										})
										if err != nil {
											return err
										}

										lock.Lock()
										nextState.Resources[id] = *res
										lock.Unlock()

										return nil
									}, backoff.NewExponentialBackOff())
								})
							}

							for _, c := range pluginClients {
								c.Kill()
							}

							// resourceOperations := map[string][]provider.Operation{}
							// for _, op := range ops {
							// 	id := op.ForIdentifier().String()
							// 	resourceOperations[id] = append(resourceOperations[id], op)
							// }
							//
							// resources := map[string]*provider.Resource{}
							// for id, operations := range resourceOperations {
							// 	sort.Slice(operations, func(i, j int) bool {
							// 		return semver.Compare(operations[i].ForVersion(), operations[j].ForVersion()) < 0
							// 	})
							//
							// 	resource := &provider.Resource{}
							// 	for _, op := range operations {
							// 		op.Apply(resource)
							// 	}
							// 	resources[id] = resource
							//
							// }
							//
							// for _, rs := range resources {
							// 	fmt.Printf("resource state: %+v\n", rs)
							// }

							return nil
						},
					},
				},
			},
			{
				Name: "provider",
				Subcommands: []*cli.Command{
					{
						Name: "generate",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "out",
								Aliases: []string{"o"},
								Value:   ".",
							},
							&cli.StringFlag{
								Name:     "mod",
								Required: true,
							},
						},
						Action: func(ctx *cli.Context) error {
							schemaPath := ctx.Args().First()
							if schemaPath == "" {
								return fmt.Errorf("must provide path to schema")
							}

							data, err := os.ReadFile(schemaPath)
							if err != nil {
								return err
							}

							p := generator.Parser{}
							schema, err := p.Parse(schemaPath, data)
							if err != nil {
								return err
							}

							outPath := filepath.Join(ctx.String("out"), schema.Name, schema.Version)
							if err := os.MkdirAll(outPath, 0777); err != nil {
								return err
							}

							g := generator.Generator{
								ModName:     ctx.String("mod"),
								ResourceDir: outPath,
							}
							for _, r := range schema.Resources {
								data, err := g.GenerateResourceIdentifier(schema.Name, schema.Version, r)
								if err != nil {
									return err
								}

								resourcePath := filepath.Join(outPath, r.Name)
								if err := os.MkdirAll(resourcePath, 0777); err != nil {
									return err
								}

								identifierPath := filepath.Join(resourcePath, "identifier.go")
								f, err := os.Create(identifierPath)
								if err != nil {
									return err
								}

								if _, err := f.Write(data); err != nil {
									return err
								}

								data, err = g.GenerateResourceOp(r)
								if err != nil {
									return err
								}

								opPath := filepath.Join(resourcePath, "op.go")
								f, err = os.Create(opPath)
								if err != nil {
									return err
								}

								if _, err := f.Write(data); err != nil {
									return err
								}

								data, err = g.GenerateClient(r)
								if err != nil {
									return err
								}

								clientPath := filepath.Join(resourcePath, "client.go")
								f, err = os.Create(clientPath)
								if err != nil {
									return err
								}

								if _, err := f.Write(data); err != nil {
									return err
								}

								resourceFilePath := filepath.Join(resourcePath, "resource.go")
								f, err = os.Create(resourceFilePath)
								if err != nil {
									return err
								}

								data, err = g.GenerateResource(r)
								if err != nil {
									return err
								}

								if _, err := f.Write(data); err != nil {
									return err
								}
							}

							providerData, err := g.GenerateProvider(schema)
							if err != nil {
								return err
							}
							providerPath := filepath.Join(outPath, "provider.go")
							f, err := os.Create(providerPath)
							if err != nil {
								return err
							}
							if _, err := f.Write(providerData); err != nil {
								return err
							}

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

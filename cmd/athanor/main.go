package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"sort"

	"github.com/alchematik/athanor/internal/parser"
	"github.com/alchematik/athanor/internal/provider"
	"github.com/alchematik/athanor/operation"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/urfave/cli/v2"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/mod/semver"
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

							var ids []any
							for providerType, p := range providers {
								for alias, provider := range p {
									fp := filepath.Join(providersPath, providerType, provider.Version, "provider.so")
									plug, err := plugin.Open(fp)
									if err != nil {
										return err
									}
									rnFuncSym, err := plug.Lookup("ResourceNames")
									if err != nil {
										return err
									}
									rnFunc, ok := rnFuncSym.(func() []string)
									if !ok {
										return fmt.Errorf("wrong type for ResourceNames symbol")
									}

									parseFuncSym, err := plug.Lookup("ParseIdentifierBlock")
									if err != nil {
										return err
									}

									parseFunc, ok := parseFuncSym.(func(*hcl.EvalContext, *hcl.Block) (any, error))
									if !ok {
										return fmt.Errorf("wrong type for ParseIdentifierBlock symbol")
									}

									resourceNames := rnFunc()
									for _, rn := range resourceNames {
										blocks := idBlocks[alias+"."+rn]
										for _, b := range blocks {
											id, err := parseFunc(evalCtx, b)
											if err != nil {
												return err
											}

											ids = append(ids, id)
										}
									}
								}
							}

							var ops []operation.Operation
							for providerType, p := range providers {
								for alias, provider := range p {
									fp := filepath.Join(providersPath, providerType, provider.Version, "provider.so")
									plug, err := plugin.Open(fp)
									if err != nil {
										return err
									}

									parseFuncSym, err := plug.Lookup("ParseOpBlock")
									if err != nil {
										return err
									}

									parseFunc, ok := parseFuncSym.(func(*hcl.EvalContext, *hcl.Block) (operation.Operation, error))
									if !ok {
										return fmt.Errorf("wrong type for ParseOpBlock symbol")
									}

									for _, b := range opBlocks[alias] {
										op, err := parseFunc(evalCtx, b)
										if err != nil {
											return err
										}

										ops = append(ops, op)
									}
								}
							}

							fmt.Printf("providers: %v\n", providers)

							migrations := map[string][]operation.Operation{}
							for _, op := range ops {
								id := op.ForIdentifier().String()
								migrations[id] = append(migrations[id], op)
							}

							resourceStates := map[string]*operation.Resource{}
							for id, resourceOps := range migrations {
								sort.Slice(resourceOps, func(i, j int) bool {
									return semver.Compare(resourceOps[i].ForVersion(), resourceOps[j].ForVersion()) < 0
								})

								for _, op := range resourceOps {
									r, ok := resourceStates[id]
									if !ok {
										r = &operation.Resource{
											Identifier: op.ForIdentifier(),
										}
										resourceStates[id] = r
									}
									op.Apply(r)
								}

							}

							for _, rs := range resourceStates {
								fmt.Printf("resource state: %+v\n", rs)
							}

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

							outPath := filepath.Join(ctx.String("out"), schema.Provider.Name, schema.Provider.Version)
							if err := os.MkdirAll(outPath, 0777); err != nil {
								return err
							}

							g := generator.Generator{
								ModName:     ctx.String("mod"),
								ResourceDir: outPath,
							}
							for _, r := range schema.Resources {
								data, err := g.GenerateResourceIdentifier(schema.Provider, r)
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

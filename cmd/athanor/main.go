package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	gcp "github.com/alchematik/athanor/gen/gcp/v0.0.1"
	"github.com/alchematik/athanor/internal/parser"
	"github.com/alchematik/athanor/internal/provider"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	// "github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/urfave/cli/v2"
	"github.com/zclconf/go-cty/cty"
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
		Type:       "op",
		LabelNames: []string{"name"},
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

							resourceNames := []string{"gcp.bucket", "gcp.bucket_object", "gcp.resource_policy"}
							idBlocks := map[string]map[string]*hcl.Block{}
							for _, r := range resourceNames {
								idBlocks[r] = map[string]*hcl.Block{}
							}

							for _, b := range blocks {
								switch b.Type {
								case "id":
									provider := b.Labels[0]
									resource := b.Labels[1]
									name := b.Labels[2]
									resourceKey := provider + "." + resource
									_, ok := idBlocks[resourceKey][name]
									if ok {
										return fmt.Errorf("dupe: %v.%v", resourceKey, name)
									}

									idBlocks[resourceKey][name] = b
								}
							}

							evalCtx := &hcl.EvalContext{
								Variables: map[string]cty.Value{
									"id": cty.ObjectVal(map[string]cty.Value{
										"gcp": cty.ObjectVal(map[string]cty.Value{
											"bucket": cty.ObjectVal(map[string]cty.Value{
												"": cty.StringVal(""),
											}),
											"bucket_object": cty.ObjectVal(map[string]cty.Value{
												"": cty.StringVal(""),
											}),
											"resource_policy": cty.ObjectVal(map[string]cty.Value{
												"": cty.StringVal(""),
											}),
										}),
									}),
								},
							}

							var ids []any
							for _, r := range resourceNames {
								blocks := idBlocks[r]
								for _, b := range blocks {
									id, err := gcp.ParseIdentifierBlock(evalCtx, b)
									if err != nil {
										return err
									}

									ids = append(ids, id)
								}
							}

							for _, id := range ids {
								fmt.Printf("ID >>> %+v, %T\n", id, id)
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

							outPath := ctx.String("out")
							if err := os.MkdirAll(outPath, 0777); err != nil {
								return err
							}

							g := generator.Generator{
								ModName:     ctx.String("mod"),
								ResourceDir: outPath,
							}
							for _, r := range schema.Resources {
								data, err := g.GenerateResourceIdentifier(r)
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

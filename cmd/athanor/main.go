package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/alchematik/athanor/internal/cli/view/deps"
	diffview "github.com/alchematik/athanor/internal/cli/view/diff"
	translatorpb "github.com/alchematik/athanor/internal/gen/go/proto/translator/v1"
	plug "github.com/alchematik/athanor/internal/plugin"

	"github.com/urfave/cli/v3"
)

func main() {
	app := cli.Command{
		Name: "athanor",
		Commands: []*cli.Command{
			{
				Name: "provider",
				Commands: []*cli.Command{
					{
						Name: "generate",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							p := cmd.Args().First()
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

							client, stop, err := translatorPlugManager.Client(c.Translator.Name, c.Translator.Version)
							if err != nil {
								return err
							}
							defer stop()

							tempFile, err := os.CreateTemp("", "")
							if err != nil {
								return err
							}

							defer os.Remove(tempFile.Name())

							_, err = client.TranslateProviderSchema(ctx, &translatorpb.TranslateProviderSchemaRequest{
								OutputPath: tempFile.Name(),
								InputPath:  c.InputPath,
							})
							if err != nil {
								return err
							}

							_, err = client.GenerateProviderSDK(ctx, &translatorpb.GenerateProviderSDKRequest{
								InputPath:  tempFile.Name(),
								OutputPath: c.OutputPath,
								Args:       c.Args,
							})
							if err != nil {
								return err
							}

							for _, clientSDK := range c.ClientSDK {
								trans, stop, err := translatorPlugManager.Client(clientSDK.Translator.Name, clientSDK.Translator.Version)
								if err != nil {
									return err
								}

								defer stop()

								_, err = trans.GenerateConsumerSDK(ctx, &translatorpb.GenerateConsumerSDKRequest{
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
				Name: "diff",
				Commands: []*cli.Command{
					{
						Name: "show",
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:  "debug",
								Usage: "log debug logs",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							program, err := diffview.NewShow(diffview.ShowParams{
								Context: ctx,
								Path:    cmd.Args().First(),
								Debug:   cmd.Bool("debug"),
							})
							_, err = program.Run()
							return err
						},
					},
					{
						Name: "reconcile",
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:  "debug",
								Usage: "log debug logs",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							program, err := diffview.NewReconcile(diffview.ShowParams{
								Context: ctx,
								Path:    cmd.Args().First(),
								Debug:   cmd.Bool("debug"),
							})
							if err != nil {
								return err
							}

							_, err = program.Run()
							return err
						},
					},
				},
			},
			{
				Name: "deps",
				Commands: []*cli.Command{
					{
						Name: "install",
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:  "debug",
								Usage: "log debug logs",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							program, err := deps.NewInstall(deps.InstallParams{
								Context: ctx,
								Path:    cmd.Args().First(),
								Debug:   cmd.Bool("debug"),
							})
							if err != nil {
								return err
							}

							_, err = program.Run()
							return err
						},
					},
				},
			},
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

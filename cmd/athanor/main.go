package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"runtime"

	"github.com/alchematik/athanor/internal/cli/view/deps"
	diffview "github.com/alchematik/athanor/internal/cli/view/diff"
	"github.com/alchematik/athanor/internal/dependency"
	plug "github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/repo"

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

							plugManager := plug.NewPlugManager(nil)
							defer plugManager.Stop()

							depManager, err := dependency.NewManager(dependency.ManagerParams{LockFilePath: "athanor.lock.json"})
							if err != nil {
								return err
							}

							binPath, err := depManager.FetchBinDependency(
								ctx,
								dependency.BinDependency{
									Type: "translator",
									Source: repo.PluginSourceLocal{
										Path: c.TranslatorsDir,
									},
									OS:   runtime.GOOS,
									Arch: runtime.GOARCH,
								},
							)

							tr, err := plugManager.Translator(binPath)
							if err != nil {
								return err
							}

							tempFile, err := os.CreateTemp("", "")
							if err != nil {
								return err
							}

							defer os.Remove(tempFile.Name())

							if err := tr.TranslateProviderSchema(ctx, c.InputPath, tempFile.Name()); err != nil {
								return err
							}

							if err := tr.GenerateProviderSDK(ctx, tempFile.Name(), c.OutputPath, c.Args); err != nil {
								return err
							}

							for _, clientSDK := range c.ClientSDK {
								path, err := depManager.FetchBinDependency(
									ctx,
									dependency.BinDependency{
										Type: "translator",
										Source: repo.PluginSourceLocal{
											Path: c.TranslatorsDir,
										},
										OS:   runtime.GOOS,
										Arch: runtime.GOARCH,
									},
								)
								if err != nil {
									return err
								}

								t, err := plugManager.Translator(path)
								if err != nil {
									return err
								}

								if err := t.GenerateConsumerSDK(ctx, tempFile.Name(), clientSDK.OutputPath); err != nil {
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
						Name: "download",
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
								Upgrade: true,
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

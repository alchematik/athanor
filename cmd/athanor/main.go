package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/alchematik/athanor/internal/cli/show"
)

func main() {
	app := cli.Command{
		Name: "athanor",
		Commands: []*cli.Command{
			{
				Name: "show",
				Commands: []*cli.Command{
					show.NewShowTargetCommand(),
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}

}

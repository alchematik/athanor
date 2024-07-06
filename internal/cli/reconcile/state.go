package reconcile

import (
	// "log/slog"
	"context"

	"github.com/alchematik/athanor/internal/cli/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v3"
)

func NewStateCommand() *cli.Command {
	return &cli.Command{
		Name: "state",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-file",
				Usage: "path to file to write logs to",
			},
			&cli.StringFlag{
				Name:  "config",
				Usage: "path to config file",
			},
		},
	}
}

func StateAction(ctx context.Context, cmd *cli.Command) error {
	// inputPath := cmd.Args().First()
	// logFilePath := cmd.String("log-file")
	// configFilePath := cmd.String("config")
	//
	// var logger *slog.Logger
	// if logFilePath != "" {
	// 	f, err := tea.LogToFile(logFilePath, "")
	// 	if err != nil {
	// 		return err
	// 	}
	//
	// 	logger = slog.New(slog.NewTextHandler(f, nil))
	// }
	m := &model.BaseModel{}
	_, err := tea.NewProgram(m).Run()
	return err
}

type StateInit struct {
}

func (m *StateInit) Init() tea.Cmd {
	return nil
}

func (m *StateInit) View() string {
	return ""
}

func (m *StateInit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// switch msg := msg.(type) {
	// default:
	// 	return m, nil
	// }
	return m, nil
}

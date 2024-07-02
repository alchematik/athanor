package show

import (
	"context"
	"log/slog"

	"github.com/alchematik/athanor/internal/cli/model"
	"github.com/alchematik/athanor/internal/scope"

	"github.com/charmbracelet/bubbles/spinner"
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
		Action: StateAction,
	}
}

func StateAction(ctx context.Context, cmd *cli.Command) error {
	inputPath := cmd.Args().First()
	logFilePath := cmd.String("log-file")
	configFilePath := cmd.String("config")

	var logger *slog.Logger
	if logFilePath != "" {
		f, err := tea.LogToFile(logFilePath, "")
		if err != nil {
			return err
		}

		logger = slog.New(slog.NewTextHandler(f, nil))
	}

	init := &StateInit{
		logger:     logger,
		inputPath:  inputPath,
		configPath: configFilePath,
		spinner:    spinner.New(),
		context:    ctx,
	}
	m := &Model{current: init, logger: logger}
	_, err := tea.NewProgram(m).Run()
	return err
}

type StateInit struct {
	logger     *slog.Logger
	inputPath  string
	configPath string
	scope      *scope.Scope
	context    context.Context
	spinner    spinner.Model
}

func (m *StateInit) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick)
}

func (m *StateInit) View() string {
	return m.spinner.View() + " initializing..."
}

func (m *StateInit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			next := &model.Quit{Logger: m.logger}
			return next, next.Init()
		}

		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}

}

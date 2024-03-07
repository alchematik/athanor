package deps

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hashicorp/go-hclog"
)

type Install struct {
	Context context.Context
	Logger  hclog.Logger
}

type InstallParams struct {
	Context context.Context
	Path    string
	Debug   bool
}

func NewInstall(params InstallParams) (*tea.Program, error) {
	logger := hclog.NewNullLogger()
	if params.Debug {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			return nil, err
		}
		logger = hclog.New(&hclog.LoggerOptions{
			Output: f,
			Level:  hclog.Debug,
		})
	}
	return tea.NewProgram(&Install{
		Context: params.Context,
		Logger:  logger,
	}), nil
}

func (m *Install) Init() tea.Cmd {
	return nil
}

func (m *Install) View() string {
	return "install!"
}

func (m *Install) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}

		return m, nil
	default:
		return m, nil
	}
}

package show

import (
	"context"

	"github.com/alchematik/athanor/internal/cli/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v3"
)

func DiffAction(ctx context.Context, cmd *cli.Command) error {
	// inputPath := cmd.Args().First()
	logFilePath := cmd.String("log-file")
	// configFilePath := cmd.String("config")

	m, err := model.NewBaseModel(logFilePath)
	if err != nil {
		return err
	}

	init := &DiffInit{}
	m.Current = init
	_, err = tea.NewProgram(m).Run()
	return err
}

type DiffInit struct {
}

func (m *DiffInit) Init() tea.Cmd {
	return nil
}

func (m *DiffInit) View() string {
	return ""
}

func (m *DiffInit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

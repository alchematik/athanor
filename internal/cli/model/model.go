package model

import (
	"log/slog"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func NewBaseModel(logFilePath string) (*BaseModel, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if logFilePath != "" {
		f, err := tea.LogToFile(logFilePath, "")
		if err != nil {
			return nil, err
		}

		logger = slog.New(slog.NewTextHandler(f, nil))
	}

	spin := spinner.New()
	return &BaseModel{
		Logger:  logger,
		Spinner: &spin,
	}, nil
}

type BaseModel struct {
	Current tea.Model
	Logger  *slog.Logger
	Spinner *spinner.Model
}

func (m *BaseModel) Init() tea.Cmd {
	return tea.Batch(m.Current.Init(), m.Spinner.Tick)
}

func (m *BaseModel) View() string {
	return m.Current.View()
}

func (m *BaseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			next := &Quit{}
			return next, next.Init()
		}

		return m, nil
	case ErrorMsg:
		next := &ErrorModel{Error: msg.Error}
		return next, next.Init()
	case spinner.TickMsg:
		spin, cmd := m.Spinner.Update(msg)
		*m.Spinner = spin
		return m, cmd
	default:
		next, cmd := m.Current.Update(msg)
		m.Current = next
		return m, cmd
	}
}

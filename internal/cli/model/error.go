package model

import (
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
)

type ErrorModel struct {
	Logger *slog.Logger
	Error  error
}

func (e *ErrorModel) Init() tea.Cmd {
	return tea.Printf("error: %s", e.Error)
}

func (e *ErrorModel) View() string {
	return e.Error.Error()
}

func (e *ErrorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return e, tea.Quit
}

type ErrorMsg struct {
	Error error
}

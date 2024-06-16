package model

import (
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
)

type Quit struct {
	Logger *slog.Logger
}

func (s *Quit) Init() tea.Cmd {
	return func() tea.Msg {
		return "quit"
	}
}

func (s *Quit) View() string {
	return "quitting..."
}

func (s *Quit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return s, tea.Quit
}

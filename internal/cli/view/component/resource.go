package component

import (
	tea "github.com/charmbracelet/bubbletea"
	// "github.com/charmbracelet/lipgloss"
)

type Resource struct {
}

func (r *Resource) Init() tea.Cmd {
	return nil
}

func (r *Resource) View() string {
	return ""
}

func (r *Resource) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return r, nil
}

package view

import (
	tea "github.com/charmbracelet/bubbletea"
)

func DisplayError(err error) tea.Msg {
	return DisplayErrorMsg{
		Error: err,
	}
}

type DisplayErrorMsg struct {
	Error error
}

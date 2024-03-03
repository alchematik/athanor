package component

import (
	"strings"

	"github.com/alchematik/athanor/internal/selector"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hashicorp/go-hclog"
)

type TreeModel struct {
	Root    *TreeNode
	Spinner spinner.Model
	Logger  hclog.Logger
}

func (m *TreeModel) Init() tea.Cmd {
	return m.Spinner.Tick
}

func (m *TreeModel) View() string {
	return m.renderStatusTreeEntry(0, "", m.Root)
}

func (m *TreeModel) renderStatusTreeEntry(space int, line string, e *TreeNode) string {
	if e == m.Root {
		var out string
		for _, entry := range e.Entries {
			out += m.renderStatusTreeEntry(0, "", entry)
		}

		return out
	}

	status := m.renderTreeNodeStatus(e.Status)
	out := status + strings.Repeat(" ", space) + line + " " + e.Kind + "/" + e.Name + "\n"
	for i, v := range e.Entries {
		line := "├─"
		if len(v.Entries) > 0 {
			line = "└─"
		}
		if i == len(e.Entries)-1 {
			line = "└─"
		}

		out += m.renderStatusTreeEntry(space+3, line, v)
	}

	return out
}

func (m *TreeModel) Update(msg tea.Msg) (*TreeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case UpdateTreeNodeMsg:
		entry := m.findEntry(msg.Selector)
		entry.Status = msg.Status
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m *TreeModel) findEntry(s selector.Selector) *TreeNode {
	if s.Parent == nil {
		for _, e := range m.Root.Entries {
			if e.Name == s.Name {
				return e
			}
		}

		return nil
	}

	parent := m.findEntry(*s.Parent)
	for _, v := range parent.Entries {
		if v.Name == s.Name {
			return v
		}
	}

	return nil
}

type TreeNode struct {
	Status  TreeNodeStatus
	Name    string
	Kind    string
	Entries []*TreeNode
}

type UpdateTreeNodeMsg struct {
	Selector selector.Selector
	Status   TreeNodeStatus
}

type TreeNodeStatus string

const (
	TreeNodeStatusLoading TreeNodeStatus = "loading"
	TreeNodeStatusUpdate                 = "update"
	TreeNodeStatusCreate                 = "create"
	TreeNodeStatusDelete                 = "delete"
	TreeNodeStatusDone                   = "done"
	TreeNodeStatusUnknown                = "unknown"
	TreeNodeStatusEmpty                  = ""
)

func (m *TreeModel) renderTreeNodeStatus(s TreeNodeStatus) string {
	switch s {
	case TreeNodeStatusLoading:
		return m.Spinner.View()
	case TreeNodeStatusCreate:
		return lipgloss.NewStyle().Foreground(ColorGreen500).Render("+")
	case TreeNodeStatusUpdate:
		return lipgloss.NewStyle().Foreground(ColorOrange500).Render("~")
	case TreeNodeStatusDelete:
		return lipgloss.NewStyle().Foreground(ColorRed500).Render("-")
	case TreeNodeStatusDone:
		return lipgloss.NewStyle().Foreground(ColorGreen400).Render("✓")
	case TreeNodeStatusUnknown:
		return lipgloss.NewStyle().Foreground(ColorOrange500).Render("?")
	default:
		return " "
	}
}

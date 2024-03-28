package component

import (
	// "fmt"
	"sort"
	"strings"

	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/selector"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hashicorp/go-hclog"
)

type DetailModel struct {
	Root    *DetailNode
	Logger  hclog.Logger
	Spinner spinner.Model
}

type DetailNode struct {
	Status TreeNodeStatus
	Name   string
	Diff   diff.Type
	Kind   string

	Entries []*DetailNode
}

func (m *DetailModel) Init() tea.Cmd {
	return nil
}

func (m *DetailModel) View() string {
	return m.renderNode(m.Root)
}

func (m *DetailModel) Update(msg tea.Msg) (*DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case UpdateDetailStatus:
		entry := m.findEntry(msg.Selector)
		entry.Status = msg.Status
		entry.Diff = msg.Diff
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m *DetailModel) renderNode(n *DetailNode) string {
	var out string

	var status string
	var style lipgloss.Style

	switch n.Status {
	case TreeNodeStatusLoading:
		status = m.Spinner.View()
	case TreeNodeStatusCreate:
		status = "+"
		style = lipgloss.NewStyle().Foreground(ColorGreen500)
	case TreeNodeStatusUpdate:
		status = "~"
		style = lipgloss.NewStyle().Foreground(ColorOrange500)
	case TreeNodeStatusDelete:
		status = "-"
		style = lipgloss.NewStyle().Foreground(ColorRed500)
	case TreeNodeStatusDone:
		status = "✓"
		style = lipgloss.NewStyle().Foreground(ColorGreen400)
	case TreeNodeStatusUnknown:
		status = "?"
		style = lipgloss.NewStyle().Foreground(ColorPurple500)
	default:
		status = " "
	}
	pad := " "
	if n.Kind != "blueprint" {
		pad += "  "
	}
	if n.Name != "" {
		out += style.Render(status+pad+n.Kind+"/"+n.Name) + "\n"

	}

	resourceDiff, isResourceDiff := n.Diff.(diff.Resource)
	if isResourceDiff && n.Status != TreeNodeStatusLoading && resourceDiff.Operation() != diff.OperationNoop {
		out += m.renderDiff(len(pad)+2, resourceDiff.ConfigDiff)
	}

	for _, e := range n.Entries {
		out += m.renderNode(e)
	}

	return out
}

func (m *DetailModel) renderDiff(spacing int, d diff.Type) string {

	var out string
	switch d := d.(type) {
	case diff.Unknown:
		out += "known after reconciliation\n"
	case diff.String:
		if d.Operation() == diff.OperationNoop {
			return out
		}

		st := diffOperationToStatus(d.Operation())

		var val string

		switch d.Operation() {
		case diff.OperationUnknown:
			val = "known after reconciliation\n"
		case diff.OperationUpdate:
			from := diffStyle(diff.OperationDelete).Render(d.From.Value)
			to := diffStyle(diff.OperationCreate).Render(d.To.Value)
			val = from + " -> " + to + "\n"
		case diff.OperationCreate:
			to := diffStyle(diff.OperationCreate).Render(d.To.Value)
			val = to + "\n"
		case diff.OperationDelete:
			from := diffStyle(diff.OperationDelete).Render(d.From.Value)
			val = from + "\n"
		}

		padding := strings.Repeat(" ", spacing)

		out += st + padding + val
	case diff.List:
		if d.Operation() == diff.OperationNoop {
			return out
		}

		for _, e := range d.Diffs {
			if e.Operation() == diff.OperationNoop {
				continue
			}

			out += m.renderDiff(spacing, e)
		}

	case diff.Map:
		if d.Operation() == diff.OperationNoop {
			return out
		}

		// var mapDiff string
		var diffs []string
		for k, v := range d.Diffs {
			var part string
			if v.Operation() == diff.OperationNoop {
				continue
			}

			style := diffStyle(v.Operation())

			st := diffOperationToStatus(v.Operation())
			part += st + strings.Repeat(" ", spacing)

			// st := diffOperationToStatus(v.Operation())
			switch v := v.(type) {
			case diff.Map:
				part += style.Render(k+":") + "\n"
				part += m.renderDiff(spacing+2, v)
			case diff.List:
				part += style.Render(k+":") + "\n"
				part += m.renderDiff(spacing+2, v)
			case diff.String:
				switch v.Operation() {
				case diff.OperationCreate:
					part += style.Render(k+"="+v.To.Value) + "\n"
				case diff.OperationUpdate:
					delStyle := diffStyle(diff.OperationDelete)
					createStyle := diffStyle(diff.OperationCreate)
					part += style.Render(k+"=") + delStyle.Render(v.From.Value) + " -> " + createStyle.Render(v.To.Value) + "\n"
				case diff.OperationDelete:
					part += style.Render(k+"="+v.From.Value) + "\n"
				}
			case diff.Unknown:
				part += style.Render(k+"=<known after reconciliation>") + "\n"
			}
			diffs = append(diffs, part)

			// val := renderDiff(spacing+1, false, v)
			//
			// out += st + " " + k + "=" + val
			// subDiff := renderDiff(spacing, false, v)
			// if subDiff != "" {
			// 	// st := diffOperationToStatus(v.Operation())
			// 	st := string(v.Operation())
			// 	mapDiff += st + k + "=" + subDiff
			// }
		}

		sort.Strings(diffs)
		for _, d := range diffs {
			out += d
		}
	}
	return out
}

func diffOperationToStatus(op diff.Operation) string {
	switch op {
	case diff.OperationCreate:
		return lipgloss.NewStyle().Foreground(ColorGreen500).Render("+")
	case diff.OperationDelete:
		return lipgloss.NewStyle().Foreground(ColorRed500).Render("-")
	case diff.OperationUpdate:
		return lipgloss.NewStyle().Foreground(ColorOrange500).Render("~")
	case diff.OperationUnknown:
		return lipgloss.NewStyle().Foreground(ColorPurple500).Render("?")
	default:
		return ""
	}
}

func diffStyle(op diff.Operation) lipgloss.Style {
	switch op {
	case diff.OperationCreate:
		return lipgloss.NewStyle().Foreground(ColorGreen500)
	case diff.OperationDelete:
		return lipgloss.NewStyle().Foreground(ColorRed500)
	case diff.OperationUpdate:
		return lipgloss.NewStyle().Foreground(ColorOrange500)
	case diff.OperationUnknown:
		return lipgloss.NewStyle().Foreground(ColorPurple500)
	default:
		return lipgloss.NewStyle()
	}
}

func (m *DetailModel) renderStatus(s TreeNodeStatus) string {
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
		return lipgloss.NewStyle().Foreground(ColorPurple500).Render("?")
	default:
		return " "
	}
}

func (m *DetailModel) findEntry(s selector.Selector) *DetailNode {
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

type UpdateDetailStatus struct {
	Name     string
	Selector selector.Selector
	Status   TreeNodeStatus
	Diff     diff.Type
}

package diff

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/evaluator"
	"github.com/alchematik/athanor/internal/spec"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Reconcile struct {
	Context         context.Context
	InputPath       string
	State           string
	Spec            spec.Spec
	TargetEvaluator *evaluator.Evaluator
	ActualEvaluator *evaluator.Evaluator
	Diff            diff.Differ
	Config          Config
	Spinner         spinner.Model
	Error           error
}

func NewReconcile() *tea.Program {
	return tea.NewProgram(&Reconcile{})
}

func (r *Reconcile) Init() tea.Cmd {
	return nil
}

func (r *Reconcile) View() string {
	switch r.State {
	case showStateInitializing:
		return "initializing..."
	case showStateInterpreting:
		return "interpreting..."
	case showStateEvaluating:
		rows := rows(0, r.Spinner, r.Diff, r.Spec, r.Diff.Result)
		str := ""
		for _, r := range rows {
			str += fmt.Sprintf("%s %s\n", r[0], r[1])
		}
		return str
	case showStateError:
		return "ERROR: " + r.Error.Error()
	default:
		return ""
	}
}

func (r *Reconcile) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return r, tea.Quit
		}

		return r, nil
	default:
		return r, nil
	}
}

package diff

import (
	api "github.com/alchematik/athanor/internal/api/resource"
	internaldiff "github.com/alchematik/athanor/internal/diff"
	plug "github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/reconcile"
	"github.com/alchematik/athanor/internal/state"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Reconcile struct {
	Tree       *Tree
	State      string
	Input      help.Model
	Reconciler *reconcile.Reconciler
}

func NewReconcile(params ShowParams) *tea.Program {
	t := NewTree(params.Context, params.Path)
	return tea.NewProgram(&Reconcile{
		State: "loading",
		Tree:  t,
		Input: help.New(),
	})
}

func (r *Reconcile) Init() tea.Cmd {
	return r.Tree.Init()
}

func (r *Reconcile) View() string {
	switch r.State {
	case "loading":
		return r.Tree.View()
	case "ready":
		t := r.Tree.View()
		m := reconcileHelpKeyMap{
			Yes: key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes")),
			No:  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "no")),
		}
		return t + "\n" + r.Input.View(m)
	default:
		return ""
	}
}

type reconcileHelpKeyMap struct {
	Yes key.Binding
	No  key.Binding
}

func (k reconcileHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Yes,
		k.No,
	}
}

func (k reconcileHelpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			k.Yes,
			k.No,
		},
	}
}

func (r *Reconcile) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// fmt.Printf(">>> %T\n", msg)
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if r.State == "ready" {
			return r, nil
		}

	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return r, tea.Quit
		}

		if r.State == "ready" {
			switch msg.String() {
			case "y":
				a := api.API{
					ProviderPluginManager: &plug.Provider{
						Dir: r.Tree.Config.ProvidersDir,
					},
				}
				e := state.Environment{
					States:        map[string]state.Type{},
					DependencyMap: map[string][]string{},
				}
				r.Reconciler = reconcile.NewReconciler(a, r.Tree.Diff.Result, e)
			case "n":
				return r, tea.Quit
			}
		}
	case doneEvaluateSpecMsg:
		if r.Tree.Diff.Result.Operation() == internaldiff.OperationNoop {
			return r, tea.Quit
		}

		r.State = "ready"
	}

	var treeCmd tea.Cmd
	r.Tree, treeCmd = r.Tree.Update(msg)
	if treeCmd != nil {
		cmds = append(cmds, treeCmd)
	}

	if len(cmds) == 0 {
		return r, nil
	}
	if len(cmds) == 1 {
		return r, cmds[0]
	}

	return r, tea.Batch(cmds...)
}

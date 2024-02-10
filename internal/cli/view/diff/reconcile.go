package diff

import (
	api "github.com/alchematik/athanor/internal/api/resource"
	"github.com/alchematik/athanor/internal/cli/view/component"
	internaldiff "github.com/alchematik/athanor/internal/diff"
	plug "github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/reconcile"
	"github.com/alchematik/athanor/internal/selector"
	"github.com/alchematik/athanor/internal/state"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hashicorp/go-hclog"
)

type Reconcile struct {
	Tree            *DiffModel
	State           string
	Input           help.Model
	Reconciler      *reconcile.Reconciler
	ReconcileQueuer *selector.Queuer
	ReconcileTree   *component.TreeModel
}

func NewReconcile(params ShowParams) *tea.Program {
	t := NewDiff(params.Context, params.Path)
	return tea.NewProgram(&Reconcile{
		State: "loading",
		Tree:  t,
		Input: help.New(),
		ReconcileTree: &component.TreeModel{
			Spinner: t.Tree.Spinner,
		},
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
	case "reconciling":
		t := r.ReconcileTree.View()
		return t
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
				p := plug.NewProvider()
				p.Dir = r.Tree.Config.ProvidersDir
				p.Logger = hclog.NewNullLogger()
				a := api.API{
					ProviderPluginManager: p,
				}
				e := state.Environment{
					States:        map[string]state.Type{},
					DependencyMap: map[string][]string{},
				}
				r.Reconciler = reconcile.NewReconciler(a, r.Tree.Differ.Result, e)

				q := selector.NewQueuer(r.Tree.Config.Name, r.Tree.Spec)
				r.ReconcileQueuer = q
				entries := componentsToEntries(r.Tree.Spec.Components)
				r.ReconcileTree.Root = &component.TreeNode{
					Entries: entries,
				}

				r.State = "reconciling"
				return r, func() tea.Msg {
					next := r.ReconcileQueuer.Next()
					return reconcileNextMsg{next: next}
				}
			case "n":
				return r, tea.Quit
			}
		}
	case doneEvaluateSpecMsg:
		if r.Tree.Differ.Result.Operation() == internaldiff.OperationNoop {
			return r, tea.Quit
		}

		r.State = "ready"
	case reconcileNextMsg:
		if len(msg.next) == 0 {
			return r, nil
		}

		var cmds []tea.Cmd
		for _, n := range msg.next {
			cmds = append(cmds, r.reconcileCmd(n))
		}

		return r, tea.Batch(cmds...)
	case setReconcileStatusMsg:
		return r, tea.Batch(func() tea.Msg {
			return component.UpdateTreeNodeMsg{
				Selector: msg.selector,
				Status:   msg.status,
			}
		}, func() tea.Msg {
			return reconcileNextMsg{
				next: r.ReconcileQueuer.Next(),
			}
		})
	}

	if r.State == "reconciling" {
		var reconcileTreeCmd tea.Cmd
		r.ReconcileTree, reconcileTreeCmd = r.ReconcileTree.Update(msg)
		if reconcileTreeCmd != nil {
			cmds = append(cmds, reconcileTreeCmd)
		}
	} else {
		var treeCmd tea.Cmd
		r.Tree, treeCmd = r.Tree.Update(msg)
		if treeCmd != nil {
			cmds = append(cmds, treeCmd)
		}
	}

	if len(cmds) == 0 {
		return r, nil
	}

	return r, tea.Batch(cmds...)
}

func (r *Reconcile) reconcileCmd(s selector.Selector) tea.Cmd {
	return func() tea.Msg {
		done, err := r.reconcile(s)
		if err != nil {
			return displayError(err)
		}

		status := component.TreeNodeStatusLoading
		if done {
			status = component.TreeNodeStatusDone
		}

		return setReconcileStatusMsg{
			selector: s,
			status:   status,
		}
	}
}

func (r *Reconcile) reconcile(s selector.Selector) (bool, error) {
	done, err := r.Reconciler.Reconcile(r.Tree.Context, s)
	if err != nil {
		return false, err
	}

	r.ReconcileQueuer.Done(s)
	return done, nil
}

type reconcileNextMsg struct {
	next []selector.Selector
}

type setReconcileStatusMsg struct {
	selector selector.Selector
	status   component.TreeNodeStatus
}

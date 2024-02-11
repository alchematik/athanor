package diff

import (
	"context"
	"sync"

	api "github.com/alchematik/athanor/internal/api/resource"
	"github.com/alchematik/athanor/internal/cli/view/component"
	"github.com/alchematik/athanor/internal/diff"
	internaldiff "github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/evaluator"
	plug "github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/reconcile"
	"github.com/alchematik/athanor/internal/selector"
	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hashicorp/go-hclog"
)

type Reconcile struct {
	Context         context.Context
	Config          Config
	Spec            spec.Spec
	State           string
	Input           help.Model
	InputPath       string
	Reconciler      *reconcile.Reconciler
	ReconcileQueuer *selector.Queuer
	ReconcileTree   *component.TreeModel
	DiffTree        *component.TreeModel
	DiffQueuer      *selector.Queuer
	Spinner         spinner.Model
	Differ          diff.Differ
	API             *api.API
}

func NewReconcile(params ShowParams) *tea.Program {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(component.ColorCyan500)
	return tea.NewProgram(&Reconcile{
		Context: params.Context,
		State:   "initializing",
		DiffTree: &component.TreeModel{
			Spinner: s,
		},
		Input:     help.New(),
		InputPath: params.Path,
		ReconcileTree: &component.TreeModel{
			Spinner: s,
		},
		Spinner: s,
	})
}

func (r *Reconcile) Init() tea.Cmd {
	return tea.Batch(r.Spinner.Tick, loadConfigCmd(r.InputPath))
}

func (r *Reconcile) View() string {
	switch r.State {
	case "initializing":
		return "initializing..."
	case "loading":
		return r.DiffTree.View()
	case "ready":
		t := r.DiffTree.View()
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
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case configLoadedMsg:
		r.Config = msg.config
		return r, translateBlueprintCmd(r.Context, r.Config)
	case setSpecMsg:
		r.Spec = msg.spec
		entries := componentsToEntries(msg.spec.Components)
		r.DiffTree.Root = &component.TreeNode{
			Entries: entries,
		}

		q := selector.NewQueuer(r.Config.Name, msg.spec)
		r.DiffQueuer = q

		target := evaluator.NewEvaluator(
			&api.Unresolved{},
			r.Spec,
			state.Environment{
				States:        map[string]state.Type{},
				DependencyMap: map[string][]string{},
			},
		)

		r.API = &api.API{
			ProviderPluginManager: plug.NewProvider(r.Config.ProvidersDir, hclog.NewNullLogger()),
		}

		actual := evaluator.NewEvaluator(
			r.API,
			r.Spec,
			state.Environment{
				States:        map[string]state.Type{},
				DependencyMap: map[string][]string{},
			},
		)
		r.Differ = diff.Differ{
			Target: target,
			Actual: actual,
			Result: diff.Environment{
				Diffs: map[string]diff.Type{},
			},
			Lock: &sync.Mutex{},
		}
		r.State = "loading"
		return r, evaluateNext(r.DiffQueuer)
	case evaluateNextMsg:
		if len(msg.next) == 0 {
			return r, nil
		}

		for _, n := range msg.next {
			cmds = append(cmds, evaluateCmd(r.Context, n, r.Differ, r.DiffQueuer))
		}
		return r, tea.Batch(cmds...)
	case setStatusMsg:
		next := r.DiffQueuer.Next()
		var cmds []tea.Cmd
		cmds = append(cmds, tea.Batch(
			func() tea.Msg { return evaluateNextMsg{next: next} },
			func() tea.Msg {
				// Initial evaluation of a blueprint is empty. Diff status is set later.
				st := msg.status
				if st == "" {
					st = "loading"
				}
				return component.UpdateTreeNodeMsg{
					Selector: msg.selector,
					Status:   component.TreeNodeStatus(st),
				}
			},
		))

		if msg.selector.Parent == nil {
			if r.Differ.Result.Diffs[msg.selector.Name].Operation() != diff.OperationEmpty {
				cmds = append(cmds, func() tea.Msg { return doneEvaluateSpec() })
			}
		}

		return r, tea.Sequence(cmds...)
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
				e := state.Environment{
					States:        map[string]state.Type{},
					DependencyMap: map[string][]string{},
				}
				r.Reconciler = reconcile.NewReconciler(r.API, r.Differ.Result, e)

				q := selector.NewQueuer(r.Config.Name, r.Spec)
				r.ReconcileQueuer = q
				entries := componentsToEntries(r.Spec.Components)
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
		if r.Differ.Result.Diffs[r.Config.Name].Operation() == internaldiff.OperationNoop {
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
		r.DiffTree, treeCmd = r.DiffTree.Update(msg)
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
	done, err := r.Reconciler.Reconcile(r.Context, s)
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

package diff

import (
	"context"

	api "github.com/alchematik/athanor/internal/api/resource"
	controller "github.com/alchematik/athanor/internal/cli/controller/diff"
	"github.com/alchematik/athanor/internal/cli/view"
	"github.com/alchematik/athanor/internal/cli/view/component"
	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/differ"
	"github.com/alchematik/athanor/internal/evaluator"
	plug "github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/reconcile"
	"github.com/alchematik/athanor/internal/selector"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hashicorp/go-hclog"
)

type Reconcile struct {
	Context       context.Context
	Config        view.Config
	State         string
	Input         help.Model
	InputPath     string
	ReconcileTree *component.TreeModel
	DiffTree      *component.TreeModel
	Spinner       spinner.Model
	API           *api.API
	Error         error

	Controller          *controller.DiffController
	ReconcileController *controller.ReconcileController

	Logger hclog.Logger
}

func NewReconcile(params ShowParams) (*tea.Program, error) {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(component.ColorCyan500)

	logger := hclog.NewNullLogger()
	if params.Debug {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			return nil, err
		}

		logger = hclog.New(&hclog.LoggerOptions{
			Output: f,
			Level:  hclog.Debug,
		})
	}
	return tea.NewProgram(&Reconcile{
		Context: params.Context,
		State:   "initializing",
		DiffTree: &component.TreeModel{
			Spinner: s,
			Logger:  logger,
		},
		Input:     help.New(),
		InputPath: params.Path,
		ReconcileTree: &component.TreeModel{
			Spinner: s,
			Logger:  logger,
		},
		Spinner: s,
		Logger:  logger,
	}), nil
}

func (r *Reconcile) Init() tea.Cmd {
	return tea.Batch(r.Spinner.Tick, view.LoadConfigCmd(r.InputPath))
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
	case "error":
		return "error: " + r.Error.Error() + "\n"
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

	switch msg := msg.(type) {
	case view.ConfigLoadedMsg:
		r.Config = msg.Config
		return r, interpretBlueprintCmd(r.Context, r.Config, r.Logger)
	case setSpecMsg:
		r.DiffTree.Root = &component.TreeNode{
			Entries: componentsToEntries(msg.spec.Spec.Components),
		}

		target := evaluator.NewEvaluator(&api.Unresolved{})

		r.API = &api.API{
			ProviderPluginManager: plug.NewProvider(r.Logger),
		}

		actual := evaluator.NewEvaluator(r.API)

		d := &differ.Differ{}

		r.Controller = controller.NewDiffController(
			r.Logger,
			msg.spec,
			target,
			actual,
			d,
		)

		r.State = "loading"
		return r, evaluateNext(r.Controller)
	case evaluateNextMsg:
		if len(msg.next) == 0 {
			return r, nil
		}

		for _, n := range msg.next {
			cmds = append(cmds, evaluateCmd(r.Logger, r.Context, r.Controller, n))
		}
		return r, tea.Batch(cmds...)
	case setStatusMsg:
		next := r.Controller.Next()
		cmds = append(cmds, tea.Batch(
			func() tea.Msg { return evaluateNextMsg{next: next} },
			func() tea.Msg {
				return component.UpdateTreeNodeMsg{
					Selector: msg.selector,
					Status:   component.TreeNodeStatus(msg.status),
				}
			},
		))

		if msg.selector.Parent == nil && msg.status != string(controller.TreeNodeStatusLoading) {
			cmds = append(cmds, func() tea.Msg { return doneEvaluateSpec() })
		}

		return r, tea.Sequence(cmds...)
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return r, tea.Quit
		}

		if r.State == "ready" {
			switch msg.String() {
			case "y":
				r.ReconcileController = controller.NewReconcileController(
					r.Logger,
					r.Controller.Spec,
					r.Controller.Diff,
					reconcile.NewReconciler(r.API),
				)

				entries := componentsToEntries(r.Controller.Spec.Spec.Components)
				r.ReconcileTree.Root = &component.TreeNode{
					Entries: entries,
				}

				r.State = "reconciling"
				return r, tea.Batch(func() tea.Msg {
					next := r.ReconcileController.Next()
					return reconcileNextMsg{next: next}
				}, r.ReconcileTree.Spinner.Tick)
			case "n":
				return r, tea.Quit
			}
		}
	case doneEvaluateSpecMsg:
		if r.Controller.Diff.Diffs[r.Config.Name].Operation() == diff.OperationNoop {
			return r, tea.Quit
		}

		r.State = "ready"
	case doneReconcilingMsg:
		return r, tea.Quit
	case reconcileNextMsg:
		if len(msg.next) == 0 {
			return r, nil
		}

		for _, n := range msg.next {
			cmds = append(cmds, r.reconcileCmd(n))
		}

		return r, tea.Batch(cmds...)
	case setReconcileStatusMsg:
		if msg.selector.Parent == nil && msg.status != component.TreeNodeStatus(controller.TreeNodeStatusLoading) {
			cmds = append(
				cmds,
				func() tea.Msg {
					return component.UpdateTreeNodeMsg{
						Selector: msg.selector,
						Status:   msg.status,
					}
				},
				func() tea.Msg { return doneReconcilingMsg{} },
			)
			return r, tea.Sequence(cmds...)
		}

		return r, tea.Batch(func() tea.Msg {
			return component.UpdateTreeNodeMsg{
				Selector: msg.selector,
				Status:   msg.status,
			}
		}, func() tea.Msg {
			return reconcileNextMsg{
				next: r.ReconcileController.Next(),
			}
		})
	case view.DisplayErrorMsg:
		r.Error = msg.Error
		r.State = "error"
		return r, quit
	case quitMsg:
		return r, tea.Quit
	}

	if len(cmds) == 0 {
		return r, nil
	}

	return r, tea.Batch(cmds...)
}

func (r *Reconcile) reconcileCmd(s selector.Selector) tea.Cmd {
	return func() tea.Msg {
		status, err := r.ReconcileController.Process(r.Context, s)
		if err != nil {
			return view.DisplayError(err)
		}

		return setReconcileStatusMsg{
			selector: s,
			status:   component.TreeNodeStatus(status),
		}
	}
}

type reconcileNextMsg struct {
	next []selector.Selector
}

type setReconcileStatusMsg struct {
	selector selector.Selector
	status   component.TreeNodeStatus
}

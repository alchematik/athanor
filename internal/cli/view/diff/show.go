package diff

import (
	"context"
	"fmt"

	api "github.com/alchematik/athanor/internal/api/resource"
	controller "github.com/alchematik/athanor/internal/cli/controller/diff"
	"github.com/alchematik/athanor/internal/cli/view"
	"github.com/alchematik/athanor/internal/cli/view/component"
	"github.com/alchematik/athanor/internal/dependency"
	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/differ"
	"github.com/alchematik/athanor/internal/evaluator"
	plug "github.com/alchematik/athanor/internal/plugin"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hashicorp/go-hclog"
)

const (
	showStateInitializing = "initializing"
	showStateTranslating  = "translating"
	showStateInterpreting = "interpreting"
	showStateEvaluating   = "evaluating"
	showStateError        = "error"
)

type Show struct {
	Context   context.Context
	Config    view.Config
	State     string
	InputPath string
	DiffTree  *component.TreeModel
	Spinner   spinner.Model
	Error     error

	Controller *controller.DiffController

	Logger hclog.Logger
}

type ShowParams struct {
	Context context.Context
	Path    string
	Debug   bool
}

func NewShow(params ShowParams) (*tea.Program, error) {
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
	return tea.NewProgram(&Show{
		Context:   params.Context,
		State:     showStateInitializing,
		InputPath: params.Path,
		DiffTree: &component.TreeModel{
			Spinner: s,
			Logger:  logger,
		},
		Logger: logger,
	}), nil
}

func (v *Show) Init() tea.Cmd {
	return tea.Batch(v.Spinner.Tick, view.LoadConfigCmd(v.InputPath))
}

func (v *Show) View() string {
	switch v.State {
	case showStateInitializing:
		return "initializing..."
	case showStateInterpreting:
		return "interpreting..."
	case showStateEvaluating:
		return v.DiffTree.View()
	case showStateError:
		return "ERROR: " + v.Error.Error() + "\n"
	default:
		return ""
	}
}

func (v *Show) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return v, tea.Quit
		}

		return v, nil
	case doneEvaluateSpecMsg:
		return v, tea.Quit
	case view.ConfigLoadedMsg:
		v.Config = msg.Config
		return v, interpretBlueprintCmd(v.Context, v.Config, v.Logger)
	case setSpecMsg:
		v.DiffTree.Root = &component.TreeNode{
			Entries: componentsToEntries(msg.spec.Spec.Components),
		}

		target := evaluator.NewEvaluator(&api.Unresolved{})

		depManager, err := dependency.NewManager(dependency.ManagerParams{
			LockFilePath: "athanor.lock.json",
		})
		if err != nil {
			return v, func() tea.Msg {
				return view.DisplayError(err)
			}
		}

		actual := evaluator.NewEvaluator(
			&api.API{
				ProviderPluginManager: plug.NewProvider(v.Logger, depManager),
			},
		)

		d := &differ.Differ{}

		v.Controller = controller.NewDiffController(
			v.Logger,
			msg.spec,
			target,
			actual,
			d,
		)

		v.State = showStateEvaluating
		return v, evaluateNext(v.Controller)
	case evaluateNextMsg:
		if len(msg.next) == 0 {
			return v, nil
		}

		var cmds []tea.Cmd
		for _, n := range msg.next {
			cmds = append(cmds, evaluateCmd(v.Logger, v.Context, v.Controller, n))
		}

		return v, tea.Batch(cmds...)
	case setStatusMsg:
		next := v.Controller.Next()
		var cmds []tea.Cmd
		cmds = append(cmds, tea.Batch(
			func() tea.Msg { return evaluateNextMsg{next: next} },
			func() tea.Msg {
				if msg.diff == nil {
					return component.UpdateTreeNodeMsg{
						Selector: msg.selector,
						Status:   component.TreeNodeStatusLoading,
					}
				} else {
					var status component.TreeNodeStatus
					switch msg.diff.Operation() {
					case diff.OperationNoop:
						status = component.TreeNodeStatusEmpty
					case diff.OperationCreate:
						status = component.TreeNodeStatusCreate
					case diff.OperationUpdate:
						status = component.TreeNodeStatusUpdate
					case diff.OperationDelete:
						status = component.TreeNodeStatusDelete
					case diff.OperationUnknown:
						status = component.TreeNodeStatusUnknown
					default:
						return view.DisplayErrorMsg{Error: fmt.Errorf("invalid diff: %v", msg.diff.Operation())}
					}

					return component.UpdateTreeNodeMsg{
						Selector: msg.selector,
						Status:   status,
					}
				}
			},
		))

		if msg.selector.Parent == nil && msg.diff != nil {
			cmds = append(cmds, func() tea.Msg { return doneEvaluateSpec() })
		}

		return v, tea.Sequence(cmds...)
	case view.DisplayErrorMsg:
		v.Error = msg.Error
		v.State = showStateError
		return v, quit
	case quitMsg:
		return v, tea.Quit
	default:
		var cmd tea.Cmd
		v.DiffTree, cmd = v.DiffTree.Update(msg)
		return v, cmd
	}
}

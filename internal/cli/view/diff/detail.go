package diff

import (
	"context"
	"fmt"
	"sort"

	api "github.com/alchematik/athanor/internal/api/resource"
	controller "github.com/alchematik/athanor/internal/cli/controller/diff"
	"github.com/alchematik/athanor/internal/cli/view"
	"github.com/alchematik/athanor/internal/cli/view/component"
	"github.com/alchematik/athanor/internal/dependency"
	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/differ"
	"github.com/alchematik/athanor/internal/evaluator"
	plug "github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/spec"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hashicorp/go-hclog"
)

type Detail struct {
	Context         context.Context
	Error           error
	Spinner         spinner.Model
	InputPath       string
	Config          view.Config
	Controller      *controller.DiffController
	State           string
	DetailViewModel *component.DetailModel

	Logger hclog.Logger
}

type DetailParams struct {
	Controller *controller.DiffController
	Context    context.Context
	Debug      bool
	Path       string
}

func NewDetail(params DetailParams) (*tea.Program, error) {
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

	return tea.NewProgram(&Detail{
		Context:   params.Context,
		InputPath: params.Path,
		Spinner:   s,
		Logger:    logger,
		DetailViewModel: &component.DetailModel{
			Logger:  logger,
			Spinner: s,
		},
	}), nil
}

func (v *Detail) Init() tea.Cmd {
	return tea.Batch(v.Spinner.Tick, view.LoadConfigCmd(v.InputPath))
}

func (v *Detail) View() string {
	if v.Error != nil {
		return v.Error.Error()
	}
	if v.DetailViewModel.Root != nil {
		return v.DetailViewModel.View()
	}

	return ""
}

func (v *Detail) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case evaluateNextMsg:
		if len(msg.next) == 0 {
			return v, nil
		}

		var cmds []tea.Cmd
		for _, n := range msg.next {
			cmds = append(cmds, evaluateCmd(v.Logger, v.Context, v.Controller, n))
		}

		return v, tea.Batch(cmds...)
	case setSpecMsg:
		v.DetailViewModel.Root = &component.DetailNode{
			Entries: componentsToDetailNode(msg.spec.Spec.Components),
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
	case setStatusMsg:
		next := v.Controller.Next()
		var cmds []tea.Cmd
		cmds = append(cmds, tea.Batch(
			func() tea.Msg { return evaluateNextMsg{next: next} },
			func() tea.Msg {
				if msg.diff == nil {
					return component.UpdateDetailStatus{
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

					return component.UpdateDetailStatus{
						Selector: msg.selector,
						Status:   status,
						Diff:     msg.diff,
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
		return v, quit
	case quitMsg:
		return v, tea.Quit
	default:
		var cmd tea.Cmd
		v.DetailViewModel, cmd = v.DetailViewModel.Update(msg)
		return v, cmd
	}
}

func componentsToDetailNode(components map[string]spec.Component) []*component.DetailNode {
	var out []*component.DetailNode
	for name, comp := range components {
		var sub []*component.DetailNode
		var kind string
		switch comp := comp.(type) {
		case spec.ComponentBuild:
			sub = componentsToDetailNode(comp.Spec.Components)
			kind = "blueprint"
		case spec.ComponentResource:
			kind = comp.Value.Identifier.ResourceType
		}
		out = append(out, &component.DetailNode{
			Kind:    kind,
			Name:    name,
			Entries: sub,
			Status:  component.TreeNodeStatusLoading,
		})
	}

	sort.Sort(detailNodeSorter(out))

	return out
}

type detailNodeSorter []*component.DetailNode

func (s detailNodeSorter) Len() int {
	return len(s)
}

func (s detailNodeSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s detailNodeSorter) Less(i, j int) bool {
	if s[i].Kind == s[j].Kind {
		return s[i].Name < s[j].Name
	}

	if s[i].Kind == "blueprint" {
		return false
	}

	if s[j].Kind == "blueprint" {
		return true
	}

	return s[i].Kind < s[j].Kind
}

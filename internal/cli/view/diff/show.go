package diff

import (
	"context"

	api "github.com/alchematik/athanor/internal/api/resource"
	controller "github.com/alchematik/athanor/internal/cli/controller/diff"
	"github.com/alchematik/athanor/internal/cli/view/component"
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
	Config    Config
	State     string
	InputPath string
	DiffTree  *component.TreeModel
	Spinner   spinner.Model
	Error     error

	Controller *controller.DiffController

	Logger hclog.Logger
}

type Config struct {
	Name       string `json:"name"`
	InputPath  string `json:"input_path"`
	Translator struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"translator"`
	TranslatorsDir string `json:"translators_dir"`
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
	return tea.Batch(v.Spinner.Tick, loadConfigCmd(v.InputPath))
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
	case configLoadedMsg:
		v.Config = msg.config
		v.State = showStateTranslating
		return v, interpretBlueprintCmd(v.Context, v.Config, v.Logger)
	case setSpecMsg:
		v.DiffTree.Root = &component.TreeNode{
			Entries: componentsToEntries(msg.spec.Spec.Components),
		}

		target := evaluator.NewEvaluator(&api.Unresolved{})

		actual := evaluator.NewEvaluator(
			&api.API{
				ProviderPluginManager: plug.NewProvider(v.Logger),
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
				return component.UpdateTreeNodeMsg{
					Selector: msg.selector,
					Status:   component.TreeNodeStatus(msg.status),
				}
			},
		))

		if msg.selector.Parent == nil && msg.status != string(controller.TreeNodeStatusLoading) {
			cmds = append(cmds, func() tea.Msg { return doneEvaluateSpec() })
		}

		return v, tea.Sequence(cmds...)
	case displayErrorMsg:
		v.Error = msg.error
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

package reconcile

import (
	"context"
	"log/slog"

	external_ast "github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/internal/cli/model"
	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/interpreter"
	"github.com/alchematik/athanor/internal/plan"
	"github.com/alchematik/athanor/internal/scope"
	"github.com/alchematik/athanor/internal/state"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v3"
)

func NewStateCommand() *cli.Command {
	return &cli.Command{
		Name: "state",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-file",
				Usage: "path to file to write logs to",
			},
			&cli.StringFlag{
				Name:  "config",
				Usage: "path to config file",
			},
		},
	}
}

func StateAction(ctx context.Context, cmd *cli.Command) error {
	inputPath := cmd.Args().First()
	logFilePath := cmd.String("log-file")

	m, err := model.NewBaseModel(logFilePath)
	if err != nil {
		return err
	}

	init := &StateInit{
		spinner:   m.Spinner,
		logger:    m.Logger,
		inputPath: inputPath,
		diff: &diff.DiffResult{
			Resources: map[string]*diff.ResourceDiff{},
			Builds:    map[string]*diff.BuildDiff{},
			Plan: &plan.Plan{
				Resources: map[string]*plan.ResourcePlan{},
				Builds:    map[string]*plan.BuildPlan{},
			},
			State: &state.State{
				Resources: map[string]*state.ResourceState{},
				Builds:    map[string]*state.BuildState{},
			},
		},
	}
	m.Current = init
	_, err = tea.NewProgram(m).Run()
	return err
}

type StateInit struct {
	logger    *slog.Logger
	spinner   *spinner.Model
	inputPath string
	scope     *scope.Scope
	diff      *diff.DiffResult
	context   context.Context
}

func (m *StateInit) Init() tea.Cmd {
	m.scope = scope.NewRootScope()
	in := &interpreter.Interpreter{Logger: m.logger}
	cmd := func() tea.Msg {
		c := diff.Converter{
			BlueprintInterpreter: in,
			PlanConverter:        &plan.Converter{BlueprintInterpreter: in},
			StateConverter:       &state.Converter{BlueprintInterpreter: in},
		}
		b := external_ast.DeclareBuild{
			Name: "Build",
			Exists: external_ast.Expr{
				Type: "bool",
				Value: external_ast.BoolLiteral{
					Value: true,
				},
			},
			Runtimeinput: external_ast.Expr{
				Value: external_ast.MapCollection{
					Value: map[string]external_ast.Expr{},
				},
			},
			BlueprintSource: external_ast.BlueprintSource{
				LocalFile: external_ast.BlueprintSourceLocalFile{
					Path: m.inputPath,
				},
			},
		}
		if _, err := c.ConvertBuildStmt(m.diff, m.scope, b); err != nil {
			return model.ErrorMsg{Error: err}
		}

		return "done"
	}
	return cmd
}

func (m *StateInit) View() string {
	return ""
}

func (m *StateInit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// switch msg := msg.(type) {
	// default:
	// 	return m, nil
	// }
	return m, nil
}

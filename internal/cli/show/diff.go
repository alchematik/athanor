package show

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

func NewDiffCommand() *cli.Command {
	return &cli.Command{
		Name: "diff",
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
		Action: DiffAction,
	}
}

func DiffAction(ctx context.Context, cmd *cli.Command) error {
	inputPath := cmd.Args().First()
	logFilePath := cmd.String("log-file")

	m, err := model.NewBaseModel(logFilePath)
	if err != nil {
		return err
	}

	init := &DiffInit{
		spinner:   m.Spinner,
		logger:    m.Logger,
		inputPath: inputPath,
		diff: &diff.Diff{
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

type DiffInit struct {
	logger    *slog.Logger
	spinner   *spinner.Model
	inputPath string
	scope     *scope.Scope
	diff      *diff.Diff
}

func (m *DiffInit) Init() tea.Cmd {
	m.scope = scope.NewScope()
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

func (m *DiffInit) View() string {
	return m.spinner.View() + " initializing..."
}

func (m *DiffInit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case string:
		return m, nil
	default:
		return m, nil
	}
}

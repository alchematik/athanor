package show

import (
	"context"
	"fmt"
	"log/slog"

	external_ast "github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/internal/cli/model"
	"github.com/alchematik/athanor/internal/convert"
	"github.com/alchematik/athanor/internal/dag"
	"github.com/alchematik/athanor/internal/eval"
	"github.com/alchematik/athanor/internal/interpreter"
	"github.com/alchematik/athanor/internal/scope"
	"github.com/alchematik/athanor/internal/state"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v3"
	"github.com/xlab/treeprint"
)

func NewShowTargetCommand() *cli.Command {
	return &cli.Command{
		Name: "target",
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
		Action: TargetAction,
	}
}

func TargetAction(ctx context.Context, cmd *cli.Command) error {
	inputPath := cmd.Args().First()
	logFilePath := cmd.String("log-file")
	configFilePath := cmd.String("config")

	initState := &Init{
		inputPath:  inputPath,
		configPath: configFilePath,
		context:    ctx,
	}
	if logFilePath != "" {
		f, err := tea.LogToFile(logFilePath, "")
		if err != nil {
			return err
		}

		initState.logger = slog.New(slog.NewTextHandler(f, nil))
	}
	_, err := tea.NewProgram(&TargetModel{current: initState}).Run()
	return err
}

type TargetModel struct {
	current tea.Model
}

func (m *TargetModel) Init() tea.Cmd {
	return m.current.Init()
}

func (m *TargetModel) View() string {
	return m.current.View()
}

func (m *TargetModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.current.Update(msg)
	m.current = next
	return m, cmd
}

type Init struct {
	logger     *slog.Logger
	inputPath  string
	configPath string
	scope      *scope.Scope
	state      *state.State
	context    context.Context
}

func (s *Init) Init() tea.Cmd {
	s.scope = scope.NewScope()
	s.state = &state.State{
		Resources: map[string]*state.ResourceState{},
		Builds:    map[string]*state.BuildState{},
	}

	return func() tea.Msg {
		c := convert.Converter{
			Logger:               s.logger,
			BlueprintInterpreter: &interpreter.Interpreter{Logger: s.logger},
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
					Path: s.inputPath,
				},
			},
		}
		if _, err := c.ConvertBuildStmt(s.state, s.scope, b); err != nil {
			return model.ErrorMsg{Error: err}
		}

		return "done"
	}
}

func (s *Init) View() string {
	return "initializing..."
}

func addNodes(t treeprint.Tree, s *state.State, build *scope.Build) {
	for _, id := range build.Resources() {
		rs, ok := s.ResourceState(id)
		if !ok {
			panic("resource not in state: " + id)
		}

		// status := rs.GetEvalState()
		exists := rs.GetExists()
		if !exists.Unknown && !exists.Value {
			continue
		}

		t.AddNode(rs.Name)
	}

	for _, id := range build.Builds() {
		bs, ok := s.BuildState(id)
		if !ok {
			panic("build not in state: " + id)
		}

		// status := bs.GetEvalState()
		exists := bs.GetExists()
		if !exists.Unknown && !exists.Value {
			continue
		}

		branch := t.AddBranch(bs.Name)

		addNodes(branch, s, build.Build(id))
	}
}

func (s *Init) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			next := &model.Quit{Logger: s.logger}
			return next, next.Init()
		}

		return s, nil
	case model.ErrorMsg:
		next := &model.ErrorModel{Logger: s.logger, Error: msg.Error}
		return next, next.Init()
	case string:
		iter := s.scope.NewIterator()
		next := &EvalModel{
			logger:    s.logger,
			state:     state.NewGlobal(s.state, nil),
			iter:      iter,
			evaluator: eval.NewTargetEvaluator(iter),
			scope:     s.scope,
			context:   s.context,
		}
		next.evaluator.Logger = s.logger

		return next, next.Init()
	default:
		return s, nil
	}
}

type EvalModel struct {
	evaluator *eval.TargetEvaluator
	state     *state.Global
	logger    *slog.Logger
	scope     *scope.Scope
	iter      *dag.Iterator
	context   context.Context
}

func (m *EvalModel) Init() tea.Cmd {
	ids := m.evaluator.Next()
	cmds := make([]tea.Cmd, len(ids))
	for i, id := range ids {
		cmds[i] = func() tea.Msg { return evalMsg{id: id} }
	}
	return tea.Batch(cmds...)
}

func (m *EvalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			next := &model.Quit{Logger: m.logger}
			return next, next.Init()
		}

		return m, nil
	case model.ErrorMsg:
		next := &model.ErrorModel{Logger: m.logger, Error: msg.Error}
		return next, next.Init()
	case evalMsg:
		return m, func() tea.Msg {
			comp, ok := m.scope.Component(msg.id)
			if !ok {
				return model.ErrorMsg{Error: fmt.Errorf("component not found: %s", msg.id)}
			}

			err := m.evaluator.Eval(m.context, m.state, comp)
			if err != nil {
				return model.ErrorMsg{Error: err}
			}

			next := m.evaluator.Next()

			return nextMsg{next: next}
		}
	case nextMsg:
		if len(msg.next) == 0 {
			return m, func() tea.Msg { return "done" }
		}
		cmds := make([]tea.Cmd, len(msg.next))
		for i, id := range msg.next {
			cmds[i] = func() tea.Msg { return evalMsg{id: id} }
		}

		return m, tea.Batch(cmds...)
	case string:
		m.logger.Info("done")
		return m, nil
	default:
		return m, nil
	}
}

func (m *EvalModel) View() string {
	tree := treeprint.New()
	build := m.scope.Build().Build(".Build")
	tree.SetValue("Build")
	addNodes(tree, m.state.Target(), build)

	return tree.String()
}

type evalMsg struct {
	id string
}

type nextMsg struct {
	next []string
}

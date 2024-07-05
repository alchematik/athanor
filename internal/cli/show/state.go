package show

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	external_ast "github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/internal/cli/model"
	"github.com/alchematik/athanor/internal/dag"
	"github.com/alchematik/athanor/internal/eval"
	"github.com/alchematik/athanor/internal/interpreter"
	"github.com/alchematik/athanor/internal/scope"
	"github.com/alchematik/athanor/internal/state"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v3"
	"github.com/xlab/treeprint"
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
		Action: StateAction,
	}
}

func StateAction(ctx context.Context, cmd *cli.Command) error {
	inputPath := cmd.Args().First()
	logFilePath := cmd.String("log-file")
	configFilePath := cmd.String("config")

	var logger *slog.Logger
	if logFilePath != "" {
		f, err := tea.LogToFile(logFilePath, "")
		if err != nil {
			return err
		}

		logger = slog.New(slog.NewTextHandler(f, nil))
	}

	init := &StateInit{
		logger:     logger,
		inputPath:  inputPath,
		configPath: configFilePath,
		spinner:    spinner.New(),
		context:    ctx,
	}
	m := &Model{current: init, logger: logger}
	_, err := tea.NewProgram(m).Run()
	return err
}

type StateInit struct {
	logger     *slog.Logger
	inputPath  string
	configPath string
	scope      *scope.Scope
	state      *state.State
	context    context.Context
	spinner    spinner.Model
}

func (m *StateInit) Init() tea.Cmd {
	m.scope = scope.NewScope()
	m.state = &state.State{
		Resources: map[string]*state.ResourceState{},
		Builds:    map[string]*state.BuildState{},
	}
	cmd := func() tea.Msg {
		c := state.Converter{
			BlueprintInterpreter: &interpreter.Interpreter{Logger: m.logger},
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
		if _, err := c.ConvertBuildStmt(m.state, m.scope, b); err != nil {
			return model.ErrorMsg{Error: err}
		}

		return "done"
	}
	return tea.Batch(m.spinner.Tick, cmd)
}

func (m *StateInit) View() string {
	return m.spinner.View() + " initializing..."
}

func (m *StateInit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case string:
		iter := m.scope.NewIterator()
		next := &StateEvalModel{
			logger:  m.logger,
			state:   m.state,
			iter:    iter,
			scope:   m.scope,
			context: m.context,
			spinner: m.spinner,
			evaluator: &eval.StateEvaluator{
				Iter:   iter,
				Logger: m.logger,
			},
		}
		return next, next.Init()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

type StateEvalModel struct {
	logger    *slog.Logger
	iter      *dag.Iterator
	context   context.Context
	spinner   spinner.Model
	state     *state.State
	scope     *scope.Scope
	evaluator *eval.StateEvaluator
}

func (s *StateEvalModel) Init() tea.Cmd {
	ids := s.evaluator.Next()
	cmds := make([]tea.Cmd, len(ids))
	for i, id := range ids {
		cmds[i] = func() tea.Msg { return evalMsg{id: id} }
	}
	cmds = append(cmds, s.spinner.Tick)
	return tea.Batch(cmds...)
}

func (s *StateEvalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case evalMsg:
		return s, func() tea.Msg {
			comp, ok := s.scope.Component(msg.id)
			if !ok {
				return model.ErrorMsg{Error: fmt.Errorf("component not found: %s", msg.id)}
			}

			err := s.evaluator.Eval(s.context, s.state, comp)
			if err != nil {
				return model.ErrorMsg{Error: err}
			}

			next := s.evaluator.Next()

			return nextMsg{next: next}
		}
	case nextMsg:
		if len(msg.next) == 0 {
			return s, func() tea.Msg { return "done" }
		}
		cmds := make([]tea.Cmd, len(msg.next))
		for i, id := range msg.next {
			cmds[i] = func() tea.Msg { return evalMsg{id: id} }
		}

		return s, tea.Batch(cmds...)
	case string:
		s.logger.Info("done")
		return s, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	default:
		return s, nil
	}
}

func (s *StateEvalModel) View() string {
	tree := treeprint.New()
	build := s.scope.Build().Build(".Build")
	b, _ := s.state.Build(".Build")
	tree.SetValue(s.renderEvalState(b.GetEvalState()) + "Build")
	s.addNodes(tree, s.state, build)

	return tree.String()
}

func (s *StateEvalModel) addNodes(t treeprint.Tree, p *state.State, build *scope.Build) {
	resources := build.Resources()
	sort.Strings(resources)
	for _, id := range resources {
		rs, ok := p.Resource(id)
		if !ok {
			panic("resource not in state: " + id)
		}

		exists := rs.GetExists()
		if !exists {
			t.AddNode(s.renderEvalState(rs.GetEvalState()) + rs.GetName())
			continue
		}

		t.AddNode(s.renderResource(rs.GetEvalState(), rs.GetName(), rs.GetResource()))
	}

	builds := build.Builds()
	sort.Strings(builds)
	for _, id := range builds {
		bs, ok := p.Build(id)
		if !ok {
			panic("build not in state: " + id)
		}

		exists := bs.GetExists()
		if !exists {
			continue
		}

		branch := t.AddBranch(s.renderEvalState(bs.GetEvalState()) + bs.GetName())

		s.addNodes(branch, p, build.Build(id))
	}
}

func (s *StateEvalModel) renderEvalState(es state.EvalState) string {
	switch es.State {
	case "", "done":
		return ""
	case "evaluating":
		return s.spinner.View()
	case "error":
		return "x"
	default:
		return ""
	}
}

func (s *StateEvalModel) renderResource(st state.EvalState, name string, r state.Resource) string {

	providerStr := fmt.Sprintf("(%s@%s)", r.Provider.Name, r.Provider.Version)

	out := s.renderEvalState(st) + name + " " + providerStr + " " + "\n"
	out += "    [identifier]\n"
	out += render(r.Identifier, 8, false)
	out += "    [config]\n"
	out += render(r.Config, 8, false)
	out += "    [attrs]\n"
	out += render(r.Attrs, 8, false)
	return out
}

func renderString(str string) string {
	return `"` + str + `"`
}

func render(val any, space int, inline bool) string {
	padding := strings.Repeat(" ", space)
	if inline {
		padding = ""
	}
	switch val := val.(type) {
	// case any:
	// 	v, ok := val.Unwrap()
	// 	if !ok {
	// 		return padding + "(known after reconcile)"
	// 	}
	//
	// 	return renderMaybe(v, space, inline)
	case string:
		return padding + renderString(val)
	case map[string]any:
		var list [][]string
		for k, v := range val {
			keyLabel := renderString(k)

			// TODO: Handle nested maps.
			list = append(list, []string{keyLabel, render(v, 0, false)})
		}

		return format(space, list)
	default:
		return fmt.Sprintf("%T", val)
	}
}

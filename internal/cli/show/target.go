package show

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	external_ast "github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/internal/cli/model"
	"github.com/alchematik/athanor/internal/convert"
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

	var logger *slog.Logger
	if logFilePath != "" {
		f, err := tea.LogToFile(logFilePath, "")
		if err != nil {
			return err
		}

		logger = slog.New(slog.NewTextHandler(f, nil))
	}

	initState := &Init{
		inputPath:  inputPath,
		configPath: configFilePath,
		context:    ctx,
		spinner:    spinner.New(),
		logger:     logger,
	}
	m := &TargetModel{current: initState, logger: logger}
	_, err := tea.NewProgram(m).Run()
	return err
}

type TargetModel struct {
	current tea.Model
	logger  *slog.Logger
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
	spinner    spinner.Model
}

func (s *Init) Init() tea.Cmd {
	s.scope = scope.NewScope()
	s.state = &state.State{
		Resources: map[string]*state.ResourceState{},
		Builds:    map[string]*state.BuildState{},
	}

	return tea.Batch(func() tea.Msg {
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
	}, s.spinner.Tick)
}

func (s *Init) View() string {
	return s.spinner.View() + " initializing..."
}

func (m *EvalModel) addNodes(t treeprint.Tree, s *state.State, build *scope.Build) {
	resources := build.Resources()
	sort.Strings(resources)
	for _, id := range resources {
		rs, ok := s.ResourceState(id)
		if !ok {
			panic("resource not in state: " + id)
		}

		exists := rs.GetExists()
		if !exists.Unknown && !exists.Value {
			t.AddNode(m.renderEvalState(rs.GetEvalState()) + rs.Name)
			continue
		}

		t.AddNode(m.renderResource(rs.GetEvalState(), rs.Name, rs.GetResource()))
	}

	builds := build.Builds()
	sort.Strings(builds)
	for _, id := range builds {
		bs, ok := s.BuildState(id)
		if !ok {
			panic("build not in state: " + id)
		}

		exists := bs.GetExists()
		if !exists.Unknown && !exists.Value {
			continue
		}

		branch := t.AddBranch(m.renderEvalState(bs.GetEvalState()) + bs.Name)

		m.addNodes(branch, s, build.Build(id))
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
			spinner:   s.spinner,
		}
		next.evaluator.Logger = s.logger

		return next, next.Init()
	case spinner.TickMsg:
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
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
	spinner   spinner.Model
}

func (m *EvalModel) Init() tea.Cmd {
	ids := m.evaluator.Next()
	cmds := make([]tea.Cmd, len(ids))
	for i, id := range ids {
		cmds[i] = func() tea.Msg { return evalMsg{id: id} }
	}
	cmds = append(cmds, m.spinner.Tick)
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
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m *EvalModel) View() string {
	tree := treeprint.New()
	build := m.scope.Build().Build(".Build")
	target := m.state.Target()
	s, _ := target.BuildState(".Build")
	tree.SetValue(m.renderEvalState(s.GetEvalState()) + "Build")
	m.addNodes(tree, m.state.Target(), build)

	return tree.String()
}

type evalMsg struct {
	id string
}

type nextMsg struct {
	next []string
}

func (m *EvalModel) renderResource(s state.EvalState, name string, r state.Maybe[state.Resource]) string {
	res, ok := r.Unwrap()
	if !ok {
		return fmt.Sprintf("%s (known after reconcile)", name)
	}

	out := m.renderEvalState(s) + name + "\n"
	out += "    [identifier]\n"
	out += render(res.Identifier, 8, false)
	out += "    [config]\n"
	out += render(res.Config, 8, false)
	out += "    [attrs]\n"
	out += render(res.Attributes, 8, false)
	return out
}

func (m *EvalModel) renderEvalState(es state.EvalState) string {
	switch es.State {
	case "", "done":
		return ""
	case "evaluating":
		return m.spinner.View()
	case "error":
		return "x"
	default:
		return ""
	}
}

const (
	unknown = "(known after reconcile)"
)

func renderString(str state.Maybe[string]) string {
	val, ok := str.Unwrap()
	if !ok {
		return unknown
	}

	return `"` + val + `"`
}

func render(val any, space int, inline bool) string {
	padding := strings.Repeat(" ", space)
	if inline {
		padding = ""
	}
	switch val := val.(type) {
	case state.Maybe[any]:
		v, ok := val.Unwrap()
		if !ok {
			return padding + "(known after reconcile)"
		}

		return render(v, space, inline)
	case state.Maybe[string]:
		return padding + renderString(val)
	case state.Maybe[map[state.Maybe[string]]state.Maybe[any]]:
		m, ok := val.Unwrap()
		if !ok {
			return padding + unknown
		}

		var list [][]string
		for k, v := range m {
			keyLabel := renderString(k)

			mapVal, ok := v.Unwrap()
			if !ok {
				list = append(list, []string{keyLabel, unknown})
				continue
			}

			// TODO: Handle nested maps.
			list = append(list, []string{keyLabel, render(mapVal, 0, false)})
		}

		return format(space, list)
	default:
		return fmt.Sprintf("%T", val)
	}
}

func format(space int, list [][]string) string {
	padding := strings.Repeat(" ", space)

	var max int
	sort.Slice(list, func(i, j int) bool {
		if len(list[i][0]) > max {
			max = len(list[i][0])
		}
		if len(list[j][0]) > max {
			max = len(list[j][0])
		}
		return list[i][0] < list[j][0]
	})

	var out string
	for _, entry := range list {
		k := entry[0]
		v := entry[1]

		if len(k) < max {
			k += strings.Repeat(" ", max-len(k))
		}

		out += padding + k + " = " + v + "\n"
	}

	return out
}

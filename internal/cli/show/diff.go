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
	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/eval"
	"github.com/alchematik/athanor/internal/interpreter"
	"github.com/alchematik/athanor/internal/plan"
	"github.com/alchematik/athanor/internal/scope"
	"github.com/alchematik/athanor/internal/state"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v3"
	"github.com/xlab/treeprint"
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

type DiffInit struct {
	logger    *slog.Logger
	spinner   *spinner.Model
	inputPath string
	scope     *scope.Scope
	diff      *diff.DiffResult
	context   context.Context
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
		iter := m.scope.NewIterator()
		next := &DiffEval{
			logger:  m.logger,
			iter:    iter,
			context: m.context,
			spinner: m.spinner,
			scope:   m.scope,
			diff:    m.diff,
			evaluator: &eval.DiffEvaluator{
				Iter:   iter,
				Logger: m.logger,
			},
		}
		return next, next.Init()
	default:
		return m, nil
	}
}

type DiffEval struct {
	logger    *slog.Logger
	iter      *dag.Iterator
	context   context.Context
	spinner   *spinner.Model
	diff      *diff.DiffResult
	scope     *scope.Scope
	evaluator *eval.DiffEvaluator
}

func (e *DiffEval) Init() tea.Cmd {
	ids := e.evaluator.Next()
	cmds := make([]tea.Cmd, len(ids))
	for i, id := range ids {
		cmds[i] = func() tea.Msg { return evalMsg{id: id} }
	}
	return tea.Batch(cmds...)
}

func (s *DiffEval) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case evalMsg:
		return s, func() tea.Msg {
			comp, ok := s.scope.Component(msg.id)
			if !ok {
				return model.ErrorMsg{Error: fmt.Errorf("component not found: %s", msg.id)}
			}

			err := s.evaluator.Eval(s.context, s.diff, comp)
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
	default:
		return s, nil
	}
}

func (s *DiffEval) View() string {
	tree := treeprint.New()
	build := s.scope.Build().Build(".Build")
	b, _ := s.diff.Build(".Build")
	tree.SetValue(s.renderEvalState(b.GetEvalState()) + "Build")
	s.addNodes(tree, s.diff, build)

	return tree.String()
}

func (s *DiffEval) renderEvalState(es diff.EvalState) string {
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

func (s *DiffEval) addNodes(t treeprint.Tree, p *diff.DiffResult, build *scope.Build) {
	resources := build.Resources()
	sort.Strings(resources)
	for _, id := range resources {
		rs, ok := p.Resource(id)
		if !ok {
			panic("resource not in state: " + id)
		}

		t.AddNode(s.renderResource(rs))
	}

	builds := build.Builds()
	sort.Strings(builds)
	for _, id := range builds {
		bs, ok := p.Build(id)
		if !ok {
			panic("build not in state: " + id)
		}

		branch := t.AddBranch(s.renderEvalState(bs.GetEvalState()) + bs.GetName())

		s.addNodes(branch, p, build.Build(id))
	}
}

func (s *DiffEval) renderResource(r *diff.ResourceDiff) string {
	p := r.GetProvider()
	providerStr := fmt.Sprintf("(%s@%s)", p.Name, p.Version)
	out := s.renderEvalState(r.GetEvalState()) + r.GetName() + " " + providerStr + " " + "\n"
	out += "    [identifier]\n"
	out += render(r.Identifier(), 8, false)
	configDiff := r.GetConfig()
	out += renderDiffAction(configDiff.Action) + "    [config]\n"
	out += s.renderDiff(configDiff, 8)
	// out += "    [attrs]\n"
	// out += render(r.Attrs, 8, false)
	return out
}

func (s *DiffEval) renderDiff(d diff.Diff[any], space int) string {
	padding := strings.Repeat(" ", space)
	switch v := d.Diff.(type) {
	case diff.Map:
		var list [][]string
		// for k, v := range d {

		// TODO: handle nested maps.
		// list = append(list, []string{k, s.renderDiff(v, 0)})
		// }
		for kd, vd := range v {
			list = append(list, []string{renderStringDiff(kd, 0), s.renderDiff(vd, 0)})
		}

		return format(space, list)
	case diff.Literal[string]:
		switch d.Action {
		case diff.ActionCreate:
			return "+ " + padding + v.Plan.Value
		case diff.ActionDelete:
			return "- " + padding + v.State.Value
		case diff.ActionUpdate:
			return "~ " + padding + fmt.Sprintf("'%s' -> '%s'", v.State.Value, v.Plan.Value)
		case diff.ActionNoop:
			return "  " + padding + v.Plan.Value
		default:
			panic("unknown action: " + d.Action)
		}
	case diff.Literal[bool]:
		switch d.Action {
		case diff.ActionCreate:
			return "+ " + padding + fmt.Sprintf("%v", v.Plan.Value)
		case diff.ActionDelete:
			return "- " + padding + fmt.Sprintf("%v", v.Plan.Value)
		case diff.ActionUpdate:
			return "~ " + padding + fmt.Sprintf("'%v' -> '%v'", v.State.Value, v.Plan.Value)
		case diff.ActionNoop:
			return "  " + padding + fmt.Sprintf("%v", v.Plan.Value)
		default:
			panic("unknown action: " + d.Action)
		}
	default:
		return fmt.Sprintf("unknown type: %T", d)
	}
}

func renderStringDiff(d diff.Diff[diff.Literal[string]], space int) string {
	padding := strings.Repeat(" ", space)
	switch d.Action {
	case diff.ActionCreate:
		return "+ " + padding + d.Diff.Plan.Value
	case diff.ActionDelete:
		return "- " + padding + d.Diff.State.Value
	case diff.ActionUpdate:
		return "~ " + padding + fmt.Sprintf("'%s' -> '%s'", d.Diff.State.Value, d.Diff.Plan.Value)
	case diff.ActionNoop:
		return "  " + padding + d.Diff.Plan.Value
	default:
		panic("unknown action: " + d.Action)
	}
}

func renderDiffAction(a diff.Action) string {
	switch a {
	case diff.ActionNoop:
		return "  "
	case diff.ActionDelete:
		return "- "
	case diff.ActionCreate:
		return "+ "
	case diff.ActionUpdate:
		return "~ "
	case diff.ActionUnknown:
		return "? "
	default:
		return "  "
	}
}

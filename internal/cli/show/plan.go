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
	"github.com/alchematik/athanor/internal/plan"
	"github.com/alchematik/athanor/internal/scope"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v3"
	"github.com/xlab/treeprint"
)

func NewPlanCommand() *cli.Command {
	return &cli.Command{
		Name: "plan",
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
		Action: PlanAction,
	}
}

func PlanAction(ctx context.Context, cmd *cli.Command) error {
	inputPath := cmd.Args().First()
	logFilePath := cmd.String("log-file")

	m, err := model.NewBaseModel(logFilePath)
	if err != nil {
		return err
	}

	init := &PlanInitModel{
		inputPath: inputPath,
		context:   ctx,
		spinner:   m.Spinner,
		logger:    m.Logger,
	}
	m.Current = init
	_, err = tea.NewProgram(m).Run()
	return err
}

type PlanInitModel struct {
	logger    *slog.Logger
	inputPath string
	scope     *scope.Scope
	plan      *plan.Plan
	context   context.Context
	spinner   *spinner.Model
}

func (s *PlanInitModel) Init() tea.Cmd {
	s.scope = scope.NewScope()
	s.plan = &plan.Plan{
		Resources: map[string]*plan.ResourcePlan{},
		Builds:    map[string]*plan.BuildPlan{},
	}

	return func() tea.Msg {
		c := plan.Converter{
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
		if _, err := c.ConvertBuildStmt(s.plan, s.scope, b); err != nil {
			return model.ErrorMsg{Error: err}
		}

		return "done"
	}
}

func (s *PlanInitModel) View() string {
	return s.spinner.View() + " initializing..."
}

func (m *PlanEvalModel) addNodes(t treeprint.Tree, p *plan.Plan, build *scope.Build) {
	resources := build.Resources()
	sort.Strings(resources)
	for _, id := range resources {
		rs, ok := p.Resource(id)
		if !ok {
			panic("resource not in state: " + id)
		}

		exists := rs.GetExists()
		if !exists.Unknown && !exists.Value {
			t.AddNode(m.renderEvalState(rs.GetEvalState()) + rs.GetName())
			continue
		}

		t.AddNode(m.renderResource(rs.GetEvalState(), rs.GetName(), rs.GetResource()))
	}

	builds := build.Builds()
	sort.Strings(builds)
	for _, id := range builds {
		bs, ok := p.Build(id)
		if !ok {
			panic("build not in state: " + id)
		}

		exists := bs.GetExists()
		if !exists.Unknown && !exists.Value {
			continue
		}

		branch := t.AddBranch(m.renderEvalState(bs.GetEvalState()) + bs.GetName())

		m.addNodes(branch, p, build.Build(id))
	}
}

func (s *PlanInitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case string:
		iter := s.scope.NewIterator()
		next := &PlanEvalModel{
			logger:    s.logger,
			plan:      s.plan,
			iter:      iter,
			evaluator: eval.NewPlanEvaluator(iter, s.logger),
			scope:     s.scope,
			context:   s.context,
			spinner:   s.spinner,
		}

		return next, next.Init()
	default:
		return s, nil
	}
}

type PlanEvalModel struct {
	evaluator *eval.PlanEvaluator
	plan      *plan.Plan
	logger    *slog.Logger
	scope     *scope.Scope
	iter      *dag.Iterator
	context   context.Context
	spinner   *spinner.Model
}

func (m *PlanEvalModel) Init() tea.Cmd {
	ids := m.evaluator.Next()
	cmds := make([]tea.Cmd, len(ids))
	for i, id := range ids {
		cmds[i] = func() tea.Msg { return evalMsg{id: id} }
	}
	return tea.Batch(cmds...)
}

func (m *PlanEvalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case evalMsg:
		return m, func() tea.Msg {
			comp, ok := m.scope.Component(msg.id)
			if !ok {
				return model.ErrorMsg{Error: fmt.Errorf("component not found: %s", msg.id)}
			}

			err := m.evaluator.Eval(m.context, m.plan, comp)
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

func (m *PlanEvalModel) View() string {
	tree := treeprint.New()
	build := m.scope.Build().Build(".Build")
	s, _ := m.plan.Build(".Build")
	tree.SetValue(m.renderEvalState(s.GetEvalState()) + "Build")
	m.addNodes(tree, m.plan, build)

	return tree.String()
}

type evalMsg struct {
	id string
}

type nextMsg struct {
	next []string
}

func (m *PlanEvalModel) renderResource(s plan.EvalState, name string, r plan.Maybe[plan.Resource]) string {
	res, ok := r.Unwrap()
	if !ok {
		return fmt.Sprintf("%s (known after reconcile)", name)
	}

	provider, ok := res.Provider.Unwrap()
	var providerStr string
	if ok {
		providerName, _ := provider.Name.Unwrap()
		providerStr += providerName
		if providerVersion, ok := provider.Version.Unwrap(); ok && providerName != "" {
			providerStr += "@" + providerVersion
		}
		providerStr = fmt.Sprintf("(%s)", providerStr)
	}

	out := m.renderEvalState(s) + name + " " + providerStr + " " + "\n"
	out += "    [identifier]\n"
	out += renderMaybe(res.Identifier, 8, false)
	out += "    [config]\n"
	out += renderMaybe(res.Config, 8, false)
	out += "    [attrs]\n"
	out += renderMaybe(res.Attrs, 8, false)
	return out
}

func (m *PlanEvalModel) renderEvalState(es plan.EvalState) string {
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

func renderMaybeString(str plan.Maybe[string]) string {
	val, ok := str.Unwrap()
	if !ok {
		return unknown
	}

	return `"` + val + `"`
}

func renderMaybe(val plan.Maybe[any], space int, inline bool) string {
	padding := strings.Repeat(" ", space)
	if inline {
		padding = ""
	}
	v, ok := val.Unwrap()
	if !ok {
		return padding + "(known after reconcile)"
	}
	switch v.(type) {
	// case plan.Maybe[any]:
	// 	v, ok := val.Unwrap()
	// 	if !ok {
	// 		return padding + "(known after reconcile)"
	// 	}
	//
	// 	return renderMaybe(v, space, inline)
	case string:
		return padding + renderMaybeString(plan.ToMaybeType[string](val))
	case map[plan.Maybe[string]]plan.Maybe[any]:
		m, ok := plan.ToMaybeType[map[plan.Maybe[string]]plan.Maybe[any]](val).Unwrap()
		if !ok {
			return padding + unknown
		}

		var list [][]string
		for k, v := range m {
			keyLabel := renderMaybeString(k)

			// TODO: Handle nested maps.
			list = append(list, []string{keyLabel, renderMaybe(v, 0, false)})
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

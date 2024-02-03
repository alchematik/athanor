package diff

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	api "github.com/alchematik/athanor/internal/api/resource"
	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/evaluator"
	consumerpb "github.com/alchematik/athanor/internal/gen/go/proto/blueprint/v1"
	translatorpb "github.com/alchematik/athanor/internal/gen/go/proto/translator/v1"
	"github.com/alchematik/athanor/internal/interpreter"
	plug "github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/selector"
	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"

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

var (
	ColorMain = ColorCyan500

	ColorCyan400 = lipgloss.Color("#7df2fe")
	ColorCyan500 = lipgloss.Color("#02daf0")
	ColorCyan600 = lipgloss.Color("#02b6c9")

	ColorGreen400 = lipgloss.Color("#9afcb3")
	ColorGreen500 = lipgloss.Color("#50fa7b")
	ColorGreen600 = lipgloss.Color("#049529")

	ColorGrey400 = lipgloss.Color("#a6a6a6")
	ColorGrey500 = lipgloss.Color("#808080")
	ColorGrey600 = lipgloss.Color("#5a5a5a")

	ColorRed400 = lipgloss.Color("#ff8080")
	ColorRed500 = lipgloss.Color("#ff3333")
	ColorRed600 = lipgloss.Color("#e60000")

	ColorOrange400 = lipgloss.Color("#ffdc9d")
	ColorOrange500 = lipgloss.Color("#ffa500")
	ColorOrange600 = lipgloss.Color("#eb9800")

	Space100 = 1
	Space200 = 2
	Space300 = 3
	Space400 = 4
	Space500 = 5
	Space600 = 6
	Space700 = 7
	Space800 = 8
)

type Show struct {
	Context         context.Context
	InputPath       string
	State           string
	Spec            spec.Spec
	TargetEvaluator *evaluator.Evaluator
	ActualEvaluator *evaluator.Evaluator
	Diff            diff.Differ
	Config          Config
	Spinner         spinner.Model
	Error           error
}

type Config struct {
	Name       string `json:"name"`
	InputPath  string `json:"input_path"`
	Translator struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"translator"`
	TranslatorsDir string `json:"translators_dir"`
	ProvidersDir   string `json:"providers_dir"`
}

type ShowParams struct {
	Context context.Context
	Path    string
}

func NewShow(params ShowParams) *tea.Program {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(ColorCyan500)

	return tea.NewProgram(&Show{
		Spinner:   s,
		Context:   params.Context,
		State:     showStateInitializing,
		InputPath: params.Path,
		Spec: spec.Spec{
			Components:    map[string]spec.Component{},
			DependencyMap: map[string][]string{},
		},
	})
}

func (v *Show) Init() tea.Cmd {
	return tea.Batch(v.Spinner.Tick, loadConfig(v.InputPath))
}

func (v *Show) View() string {
	switch v.State {
	case showStateInitializing:
		return "initializing..."
	case showStateInterpreting:
		return "interpreting..."
	case showStateEvaluating:
		rows := v.rows(0, v.Spec, v.Diff.Result)
		str := ""
		for _, r := range rows {
			str += fmt.Sprintf("%s %s\n", r[0], r[1])
		}
		return str
	case showStateError:
		return "ERROR: " + v.Error.Error()
	default:
		return ""
	}
}

func (v *Show) rows(spaces int, s spec.Spec, envDiff diff.Environment) [][]string {
	resourceToAlias := map[string][]string{}
	var builds []string
	for k, comp := range s.Components {
		switch comp := comp.(type) {
		case spec.ComponentResource:
			rt := comp.Value.Identifier.ResourceType
			resourceToAlias[rt] = append(resourceToAlias[rt], k)
		case spec.ComponentBuild:
			builds = append(builds, k)
		}
	}

	resourceTypes := make([]string, 0, len(resourceToAlias))
	for k := range resourceToAlias {
		resourceTypes = append(resourceTypes, k)
	}

	sort.Strings(resourceTypes)

	sort.Strings(builds)

	var out [][]string
	for i, rt := range resourceTypes {
		resources := resourceToAlias[rt]
		sort.Strings(resources)
		for j, r := range resources {
			st := v.Spinner.View()
			v.Diff.Lock.Lock()
			if d, ok := envDiff.Diffs[r]; ok {
				st = sign(d.Operation())
			}
			v.Diff.Lock.Unlock()
			tree := "├─"
			if i == len(resourceTypes)-1 && j == len(resources)-1 && len(builds) == 0 {
				tree = "└─"
			}
			out = append(out, []string{st, strings.Repeat(" ", spaces) + tree + " " + rt + "/" + r})
		}
	}

	for i, b := range builds {
		build := s.Components[b].(spec.ComponentBuild)

		st := v.Spinner.View()
		d, ok := envDiff.Diffs[b]
		if ok && d.Operation() != diff.OperationEmpty {
			st = sign(d.Operation())
		}

		subEnv, ok := d.(diff.Environment)
		if !ok {
			subEnv = diff.Environment{}
		}

		tree := ""
		if spaces > 0 {
			tree = "├─"
			if i == len(builds)-1 {
				tree = "└─"
			}
		}

		out = append(out, []string{st, strings.Repeat(" ", spaces) + tree + "blueprint" + "/" + b})
		sub := v.rows(spaces+1, build.Spec, subEnv)
		out = append(out, sub...)
	}

	return out
}

func sign(op diff.Operation) string {
	switch op {
	case diff.OperationEmpty:
		return " "
	case diff.OperationUpdate:
		return lipgloss.NewStyle().Foreground(ColorOrange500).Render("~")
	case diff.OperationCreate:
		return lipgloss.NewStyle().Foreground(ColorGreen500).Render("+")
	case diff.OperationDelete:
		return lipgloss.NewStyle().Foreground(ColorRed500).Render("-")
	case diff.OperationNoop:
		return " "
	case diff.OperationUnknown:
		return "?"
	default:
		return " "
	}
}

func (v *Show) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return v, tea.Quit
		}

		return v, nil
	case setConfigMsg:
		v.Config = msg.config
		v.State = showStateTranslating
		return v, translateBlueprint(
			v.Context,
			v.Config,
		)
	case setSpecMsg:
		v.Spec = msg.spec
		v.TargetEvaluator = evaluator.NewEvaluator(
			&api.Unresolved{},
			v.Spec,
			state.Environment{
				States:        map[string]state.Type{},
				DependencyMap: map[string][]string{},
			},
		)

		p := plug.NewProvider()
		p.Dir = v.Config.ProvidersDir
		p.Logger = hclog.NewNullLogger()
		v.ActualEvaluator = evaluator.NewEvaluator(
			&api.API{
				ProviderPluginManager: p,
			},
			v.Spec,
			state.Environment{
				States:        map[string]state.Type{},
				DependencyMap: map[string][]string{},
			},
		)
		v.Diff = diff.Differ{
			Target: v.TargetEvaluator.Env,
			Actual: v.ActualEvaluator.Env,
			Result: diff.Environment{
				Diffs: map[string]diff.Type{},
			},
			Lock: &sync.Mutex{},
		}
		v.State = showStateEvaluating
		return v, evaluateNext(v.ActualEvaluator)
	case evaluateNextMsg:
		if len(msg.next) == 0 {
			return v, nil
		}

		var cmds []tea.Cmd
		for _, n := range msg.next {
			cmds = append(cmds, evaluate(v.Context, n, v.TargetEvaluator, v.ActualEvaluator, v.Diff))
		}
		return v, tea.Batch(cmds...)
	case doneEvaluateSpecMsg:
		return v, tea.Quit
	case displayErrorMsg:
		v.Error = msg.error
		v.State = showStateError
		return v, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		v.Spinner, cmd = v.Spinner.Update(msg)
		return v, cmd
	default:
		return v, nil
	}
}

func evaluate(ctx context.Context, s selector.Selector, target, actual *evaluator.Evaluator, d diff.Differ) tea.Cmd {
	return func() tea.Msg {
		if err := target.Eval(ctx, s); err != nil {
			return displayError(err)
		}

		if err := actual.Eval(ctx, s); err != nil {
			return displayError(err)
		}

		if err := d.Diff(s); err != nil {
			return displayError(err)
		}

		if s.Parent == nil {
			if d.Result.Diffs[s.Name].Operation() != diff.OperationEmpty {
				return doneEvaluateSpec()
			}
		}

		return evaluateNextMsg{next: actual.Next()}
	}
}

func loadConfig(inputPath string) tea.Cmd {
	return func() tea.Msg {
		f, err := os.ReadFile(inputPath)
		if err != nil {
			return displayError(err)
		}

		var c Config
		if err := json.Unmarshal(f, &c); err != nil {
			return displayError(err)
		}

		return setConfigMsg{config: c}
	}
}

type setConfigMsg struct {
	config Config
}

func displayError(err error) tea.Msg {
	return displayErrorMsg{
		error: err,
	}
}

type displayErrorMsg struct {
	error error
}

func translateBlueprint(ctx context.Context, config Config) tea.Cmd {
	return func() tea.Msg {
		translatorPlugManager := plug.Translator{
			Dir: config.TranslatorsDir,
		}

		client, stop, err := translatorPlugManager.Client(config.Translator.Name, config.Translator.Version)
		if err != nil {
			return displayError(fmt.Errorf("error getting translation client: %v", err))
		}
		defer stop()

		tempFile, err := os.CreateTemp("", "")
		if err != nil {
			return displayError(err)
		}

		defer os.Remove(tempFile.Name())

		_, err = client.TranslateBlueprint(ctx, &translatorpb.TranslateBlueprintRequest{
			InputPath:  config.InputPath,
			OutputPath: tempFile.Name(),
		})
		if err != nil {
			return displayError(fmt.Errorf("error translating blueprint: %v", err))
		}

		blueprintData, err := os.ReadFile(tempFile.Name())
		if err != nil {
			return displayError(err)
		}

		var blueprint consumerpb.Blueprint
		if err := json.Unmarshal(blueprintData, &blueprint); err != nil {
			return displayError(fmt.Errorf("error unmarshaling blueprint: %v", err))
		}

		bp, err := convertBlueprint(&blueprint)
		if err != nil {
			return displayError(fmt.Errorf("error converting blueprint: %v", err))
		}

		in := interpreter.Interpreter{}
		s := spec.Spec{
			Components:    map[string]spec.Component{},
			DependencyMap: map[string][]string{},
		}
		if err := in.Interpret(ctx, s, ast.StmtBuild{
			Alias: config.Name,
			Blueprint: ast.ExprBlueprint{
				Stmts: bp.Stmts,
			},
		}); err != nil {
			return displayError(err)
		}

		return setSpecMsg{
			spec: s,
		}
	}
}

type setSpecMsg struct {
	spec spec.Spec
}

func evaluateNext(eval *evaluator.Evaluator) tea.Cmd {
	return func() tea.Msg {
		return evaluateNextMsg{
			next: eval.Next(),
		}
	}
}

type evaluateNextMsg struct {
	next []selector.Selector
}

func doneEvaluateSpec() tea.Msg {
	return doneEvaluateSpecMsg{}
}

type doneEvaluateSpecMsg struct {
}

func convertBlueprint(bp *consumerpb.Blueprint) (ast.Blueprint, error) {
	out := ast.Blueprint{}
	for _, stmt := range bp.GetStmts() {
		converted, err := convertStmt(stmt)
		if err != nil {
			return ast.Blueprint{}, err
		}

		out.Stmts = append(out.Stmts, converted)
	}

	return out, nil
}

func convertStmt(st *consumerpb.Stmt) (ast.Stmt, error) {
	switch s := st.GetType().(type) {
	case *consumerpb.Stmt_Resource:
		ex, err := convertExpr(s.Resource.GetExpr())
		if err != nil {
			return nil, err
		}

		return ast.StmtResource{
			Expr: ex,
		}, nil
	case *consumerpb.Stmt_Build:
		ex, err := convertExpr(s.Build.GetBlueprint())
		if err != nil {
			return nil, err
		}

		inputs := map[string]ast.Expr{}
		for name, inputExpr := range s.Build.GetInputs() {
			input, err := convertExpr(inputExpr)
			if err != nil {
				return nil, err
			}

			inputs[name] = input
		}

		return ast.StmtBuild{
			Alias:     s.Build.GetAlias(),
			Blueprint: ex,
			Inputs:    inputs,
		}, nil
	default:
		return nil, fmt.Errorf("invalid stmt: %T", st.GetType())
	}
}

func convertExpr(ex *consumerpb.Expr) (ast.Expr, error) {
	switch e := ex.GetType().(type) {
	case *consumerpb.Expr_Blueprint:
		stmts := make([]ast.Stmt, len(e.Blueprint.GetStmts()))
		for i, s := range e.Blueprint.GetStmts() {
			converted, err := convertStmt(s)
			if err != nil {
				return nil, err
			}

			stmts[i] = converted
		}

		return ast.ExprBlueprint{Stmts: stmts}, nil
	case *consumerpb.Expr_Provider:
		return ast.ExprProvider{
			Name:    e.Provider.GetName(),
			Version: e.Provider.GetVersion(),
		}, nil
	case *consumerpb.Expr_Resource:
		provider, err := convertExpr(e.Resource.GetProvider())
		if err != nil {
			return nil, err
		}

		id, err := convertExpr(e.Resource.GetIdentifier())
		if err != nil {
			return nil, err
		}

		config, err := convertExpr(e.Resource.GetConfig())
		if err != nil {
			return nil, err
		}

		exists, err := convertExpr(e.Resource.GetExists())
		if err != nil {
			return nil, err
		}

		return ast.ExprResource{
			Provider:   provider,
			Identifier: id,
			Config:     config,
			Exists:     exists,
		}, nil
	case *consumerpb.Expr_ResourceIdentifier:
		val, err := convertExpr(e.ResourceIdentifier.GetValue())
		if err != nil {
			return ast.ExprResourceIdentifier{}, err
		}

		return ast.ExprResourceIdentifier{
			Alias:        e.ResourceIdentifier.GetAlias(),
			ResourceType: e.ResourceIdentifier.GetType(),
			Value:        val,
		}, nil
	case *consumerpb.Expr_StringLiteral:
		return ast.ExprString{Value: e.StringLiteral}, nil
	case *consumerpb.Expr_BoolLiteral:
		return ast.ExprBool{Value: e.BoolLiteral}, nil
	case *consumerpb.Expr_File:
		return ast.ExprFile{Path: e.File.Path}, nil
	case *consumerpb.Expr_Map:
		m := ast.ExprMap{Entries: map[string]ast.Expr{}}
		for k, v := range e.Map.GetEntries() {
			var err error
			m.Entries[k], err = convertExpr(v)
			if err != nil {
				return nil, err
			}
		}

		return m, nil
	case *consumerpb.Expr_List:
		l := make([]ast.Expr, len(e.List.Elements))
		for i, val := range e.List.Elements {
			converted, err := convertExpr(val)
			if err != nil {
				return nil, err
			}
			l[i] = converted
		}

		return ast.ExprList{
			Elements: l,
		}, nil
	case *consumerpb.Expr_Get:
		obj, err := convertExpr(e.Get.GetObject())
		if err != nil {
			return nil, err
		}

		g := ast.ExprGet{
			Name:   e.Get.GetName(),
			Object: obj,
		}

		return g, nil
	case *consumerpb.Expr_Nil:
		return ast.ExprNil{}, nil
	default:
		return nil, fmt.Errorf("invalid expr: %T", ex.GetType())
	}
}

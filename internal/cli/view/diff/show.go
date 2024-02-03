package diff

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	// "time"

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
	resourceHeaderStyle = lipgloss.NewStyle().Bold(true)

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
	Model *ShowModel
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

type ShowModel struct {
	Context         context.Context
	InputPath       string
	BlueprintName   string
	State           string
	Spec            spec.Spec
	Error           error
	TargetEvaluator *evaluator.Evaluator
	ActualEvaluator *evaluator.Evaluator
	Diff            diff.Differ
	Config          Config
	Spinner         spinner.Model
}

func NewShow(params ShowParams) *tea.Program {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(ColorCyan500)

	return tea.NewProgram(&Show{
		Model: &ShowModel{
			Spinner:   s,
			Context:   params.Context,
			State:     showStateInitializing,
			InputPath: params.Path,
			Spec: spec.Spec{
				Components:    map[string]spec.Component{},
				DependencyMap: map[string][]string{},
			},
		},
	})
}

func (v *Show) Init() tea.Cmd {
	return v.Model.Spinner.Tick
}

func (v *Show) View() string {
	switch v.Model.State {
	case showStateInitializing:
		return "initializing..."
	case showStateInterpreting:
		return "interpreting..."
	case showStateEvaluating:
		rows := v.rows(0, v.Model.Spec, v.Model.Diff.Result)
		str := ""
		for _, r := range rows {
			str += fmt.Sprintf("%s %s\n", r[0], r[1])
		}
		return str
	case "done":
		rows := v.rows(0, v.Model.Spec, v.Model.Diff.Result)
		str := ""
		for _, r := range rows {
			str += fmt.Sprintf("%s %s\n", r[0], r[1])
		}
		return str
	case showStateError:
		return "ERROR: " + v.Model.Error.Error()
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
			st := v.Model.Spinner.View()
			// st := "loading"
			v.Model.Diff.Lock.Lock()
			if d, ok := envDiff.Diffs[r]; ok {
				st = sign(d.Operation())
				// fmt.Printf(">> %v\n", st)
			}
			v.Model.Diff.Lock.Unlock()
			tree := "├─"
			if i == len(resourceTypes)-1 && j == len(resources)-1 && len(builds) == 0 {
				tree = "└─"
			}
			out = append(out, []string{st, strings.Repeat(" ", spaces) + tree + " " + rt + "/" + r})
		}
	}

	for i, b := range builds {
		build := s.Components[b].(spec.ComponentBuild)

		st := v.Model.Spinner.View()
		// st := "loading"
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
	// if _, ok := msg.(spinner.TickMsg); !ok {
	// 	fmt.Printf("UPDATE: %T, %+v\n", msg, msg)
	// }
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if v.Model.State == showStateInitializing {
			return v, loadConfig
		}

		return v, nil
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return v, tea.Quit
		}

		return v, nil
	case loadConfigMsg:
		return v, func() tea.Msg {
			f, err := os.ReadFile(v.Model.InputPath)
			if err != nil {
				return displayError(err)
			}

			var c Config
			if err := json.Unmarshal(f, &c); err != nil {
				return displayError(err)
			}

			v.Model.BlueprintName = c.Name
			v.Model.State = showStateTranslating
			v.Model.Config = c
			return translateBlueprint(c.InputPath, c.Translator.Name, c.Translator.Version, c.TranslatorsDir)
		}
	case translateBlueprintMsg:
		return v, func() tea.Msg {
			translatorPlugManager := plug.Translator{
				Dir: msg.dir,
			}

			client, stop, err := translatorPlugManager.Client(msg.name, msg.version)
			if err != nil {
				return displayError(fmt.Errorf("error getting translation client: %v", err))
			}
			defer stop()

			tempFile, err := os.CreateTemp("", "")
			if err != nil {
				return displayError(err)
			}

			defer os.Remove(tempFile.Name())

			_, err = client.TranslateBlueprint(v.Model.Context, &translatorpb.TranslateBlueprintRequest{
				InputPath:  msg.path,
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

			v.Model.State = showStateInterpreting
			return interpretBlueprint(bp)
		}
	case interpretBlueprintMsg:
		return v, func() tea.Msg {
			in := interpreter.Interpreter{}
			if err := in.Interpret(v.Model.Context, v.Model.Spec, ast.StmtBuild{
				Alias: v.Model.BlueprintName,
				Blueprint: ast.ExprBlueprint{
					Stmts: msg.blueprint.Stmts,
				},
			}); err != nil {
				return displayError(err)
			}

			v.Model.TargetEvaluator = evaluator.NewEvaluator(
				&api.Unresolved{},
				v.Model.Spec,
				state.Environment{
					States:        map[string]state.Type{},
					DependencyMap: map[string][]string{},
				},
			)

			p := plug.NewProvider()
			p.Dir = v.Model.Config.ProvidersDir
			p.Logger = hclog.NewNullLogger()
			v.Model.ActualEvaluator = evaluator.NewEvaluator(
				&api.API{
					ProviderPluginManager: p,
				},
				v.Model.Spec,
				state.Environment{
					States:        map[string]state.Type{},
					DependencyMap: map[string][]string{},
				},
			)
			v.Model.Diff = diff.Differ{
				Target: v.Model.TargetEvaluator.Env,
				Actual: v.Model.ActualEvaluator.Env,
				Result: diff.Environment{
					Diffs: map[string]diff.Type{},
				},
				Lock: &sync.Mutex{},
			}
			v.Model.State = showStateEvaluating

			return startEvaluateSpec()
		}
	case startEvaluateSpecMsg:
		next := v.Model.ActualEvaluator.Next()
		// fmt.Printf("NEXT: %v\n", next)
		if len(next) == 0 {
			return v, nil
		}

		var cmds []tea.Cmd
		for _, n := range next {
			cmds = append(cmds, evaluate(n))
		}
		return v, tea.Batch(cmds...)
	case evaluateMsg:
		return v, tea.Sequence(func() tea.Msg {
			// fmt.Printf("START EVAL >>>>>>>> %v\n", msg)
			if err := v.Model.TargetEvaluator.Eval(v.Model.Context, msg.selector); err != nil {
				return displayError(err)
			}

			// start := time.Now()
			if err := v.Model.ActualEvaluator.Eval(v.Model.Context, msg.selector); err != nil {
				return displayError(err)
			}
			// fmt.Printf("TIME: %v\n", time.Since(start))

			// fmt.Printf("DONE EVAL >>>>>>>> %v\n", msg)

			return doDiff(msg.selector)
		}, startEvaluateSpec)
	case diffMsg:
		return v, func() tea.Msg {
			if err := v.Model.Diff.Diff(msg.selector); err != nil {
				return displayError(err)
			}

			// fmt.Printf("DONE >>>>>>>> %v\n", msg)
			if msg.selector.Parent == nil {
				if v.Model.Diff.Result.Diffs[msg.selector.Name].Operation() != diff.OperationEmpty {
					return doneEvaluateSpec()
				}
			}

			return nil
		}
	case doneEvaluateSpecMsg:
		return v, tea.Quit
		// return v, nil
	case exitMsg:
		return v, tea.Quit
	case displayErrorMsg:
		v.Model.Error = msg.error
		v.Model.State = showStateError
		// return v, tea.Quit
		return v, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		v.Model.Spinner, cmd = v.Model.Spinner.Update(msg)
		return v, cmd
	default:
		return v, nil
	}
}

type exitMsg struct {
}

type diffMsg struct {
	selector selector.Selector
}

func doDiff(s selector.Selector) tea.Msg {
	return diffMsg{selector: s}
}

type evaluateMsg struct {
	selector selector.Selector
}

func evaluate(s selector.Selector) tea.Cmd {
	return func() tea.Msg {
		return evaluateMsg{selector: s}
	}
}

func loadConfig() tea.Msg {
	return loadConfigMsg{}
}

type loadConfigMsg struct{}

func displayError(err error) tea.Msg {
	return displayErrorMsg{
		error: err,
	}
}

type displayErrorMsg struct {
	error error
}

func translateBlueprint(inputPath, name, version, dir string) tea.Msg {
	return translateBlueprintMsg{
		path:    inputPath,
		name:    name,
		version: version,
		dir:     dir,
	}
}

type translateBlueprintMsg struct {
	path    string
	name    string
	version string
	dir     string
}

func interpretBlueprint(bp ast.Blueprint) tea.Msg {
	return interpretBlueprintMsg{
		blueprint: bp,
	}
}

type interpretBlueprintMsg struct {
	blueprint ast.Blueprint
}

func startEvaluateSpec() tea.Msg {
	return startEvaluateSpecMsg{}
}

type startEvaluateSpecMsg struct {
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

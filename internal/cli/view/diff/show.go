package diff

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	api "github.com/alchematik/athanor/internal/api/resource"
	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/evaluator"
	consumerpb "github.com/alchematik/athanor/internal/gen/go/proto/blueprint/v1"
	translatorpb "github.com/alchematik/athanor/internal/gen/go/proto/translator/v1"
	"github.com/alchematik/athanor/internal/interpreter"
	plug "github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/spec"
	"github.com/alchematik/athanor/internal/state"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

const (
	showStateInitializing = "initializing"
	showStateTranslating  = "translating"
	showStateInterpreting = "interpreting"
	showStateEvaluating   = "evaluating"
	showStateError        = "error"
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
}

func NewShow(params ShowParams) *tea.Program {
	return tea.NewProgram(&Show{
		Model: &ShowModel{
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
	return nil
}

func (v *Show) View() string {
	switch v.Model.State {
	case showStateInitializing:
		return "initializing..."
	case showStateInterpreting:
		return "interpreting..."
	case showStateEvaluating:
		rows := v.rows(0, v.Model.Spec, v.Model.TargetEvaluator.Env)
		t := table.New().
			Border(lipgloss.HiddenBorder()).
			Rows(rows...)

		return t.String()
	case showStateError:
		return "ERROR: " + v.Model.Error.Error()
	default:
		return ""
	}
}

func (v *Show) rows(spaces int, s spec.Spec, e state.Environment) [][]string {
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
	for _, rt := range resourceTypes {
		for _, r := range resourceToAlias[rt] {
			state := "loading..."
			if _, ok := e.States[r]; ok {
				state = "done!"
			}
			out = append(out, []string{state, strings.Repeat("  ", spaces) + rt + "/" + r})
		}
	}

	for _, b := range builds {
		build := s.Components[b].(spec.ComponentBuild)
		env, ok := e.States[b].(state.Environment)
		state := "loading..."
		if ok && env.Done {
			state = "done!"
		}
		out = append(out, []string{state, strings.Repeat("  ", spaces) + "blueprint" + "/" + b})
		sub := v.rows(spaces+1, build.Spec, env)
		out = append(out, sub...)
	}

	return out
}

func (v *Show) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		f, err := os.ReadFile(v.Model.InputPath)
		if err != nil {
			return v, displayError(err)
		}

		var c Config
		if err := json.Unmarshal(f, &c); err != nil {
			return v, displayError(err)
		}

		v.Model.BlueprintName = c.Name
		v.Model.State = showStateTranslating
		return v, translateBlueprint(c.InputPath, c.Translator.Name, c.Translator.Version, c.TranslatorsDir)
	case translateBlueprintMsg:
		translatorPlugManager := plug.Translator{
			Dir: msg.dir,
		}

		client, stop, err := translatorPlugManager.Client(msg.name, msg.version)
		if err != nil {
			return v, displayError(fmt.Errorf("error getting translation client: %v", err))
		}
		defer stop()

		tempFile, err := os.CreateTemp("", "")
		if err != nil {
			return v, displayError(err)
		}

		defer os.Remove(tempFile.Name())

		_, err = client.TranslateBlueprint(v.Model.Context, &translatorpb.TranslateBlueprintRequest{
			InputPath:  msg.path,
			OutputPath: tempFile.Name(),
		})
		if err != nil {
			return v, displayError(fmt.Errorf("error translating blueprint: %v", err))
		}

		blueprintData, err := os.ReadFile(tempFile.Name())
		if err != nil {
			return v, displayError(err)
		}

		var blueprint consumerpb.Blueprint
		if err := json.Unmarshal(blueprintData, &blueprint); err != nil {
			return v, displayError(fmt.Errorf("error unmarshaling blueprint: %v", err))
		}

		bp, err := convertBlueprint(&blueprint)
		if err != nil {
			return v, displayError(fmt.Errorf("error converting blueprint: %v", err))
		}

		v.Model.State = showStateInterpreting
		return v, interpretBlueprint(bp)
	case interpretBlueprintMsg:
		in := interpreter.Interpreter{}
		if err := in.Interpret(v.Model.Context, v.Model.Spec, ast.StmtBuild{
			Alias: v.Model.BlueprintName,
			Blueprint: ast.ExprBlueprint{
				Stmts: msg.blueprint.Stmts,
			},
		}); err != nil {
			return v, displayError(err)
		}

		// eval := evaluator.Evaluator{
		// 	ResourceAPI: &api.Unresolved{},
		// }

		// desiredEnv, err := eval.Evaluate(v.Model.Context, spec)
		// if err != nil {
		// 	return v, displayError(err)
		// }
		//
		// // v.Model.Environment = desiredEnv
		// v.Model.TargetEvaluator = evaluator.Evaluator{
		// 	ResourceAPI: &api.Unresolved{},
		// 	Spec:        v.Model.Spec,
		// 	Env: state.Environment{
		// 		States:        map[string]state.Type{},
		// 		DependencyMap: map[string][]string{},
		// 	},
		// }
		v.Model.TargetEvaluator = evaluator.NewEvaluator(
			&api.Unresolved{},
			v.Model.Spec,
			state.Environment{
				States:        map[string]state.Type{},
				DependencyMap: map[string][]string{},
			},
		)
		v.Model.State = showStateEvaluating

		return v, startEvaluateSpec()
	case startEvaluateSpecMsg:
		next := v.Model.TargetEvaluator.Next()
		// log.Printf("NEXT >>>>> %v\n", next)
		if len(next) == 0 {
			return v, doneEvaluateSpec()
		}

		var cmds []tea.Cmd
		for _, n := range next {
			cmds = append(cmds, evaluate(n))
		}
		return v, tea.Batch(cmds...)
	case evaluateMsg:
		if err := v.Model.TargetEvaluator.Eval(v.Model.Context, msg.selector); err != nil {
			return v, displayError(err)
		}

		return v, startEvaluateSpec()
	case doneEvaluateSpecMsg:
		return v, nil

	case displayErrorMsg:
		v.Model.Error = msg.error
		v.Model.State = showStateError
		// return v, tea.Quit
		return v, nil
	default:
		return v, nil
	}
}

type evaluateMsg struct {
	selector evaluator.Selector
}

func evaluate(s evaluator.Selector) tea.Cmd {
	return func() tea.Msg {
		return evaluateMsg{selector: s}
	}
}

func loadConfig() tea.Msg {
	return loadConfigMsg{}
}

type loadConfigMsg struct{}

func displayError(err error) tea.Cmd {
	return func() tea.Msg {
		return displayErrorMsg{
			error: err,
		}
	}
}

type displayErrorMsg struct {
	error error
}

func translateBlueprint(inputPath, name, version, dir string) tea.Cmd {
	return func() tea.Msg {
		return translateBlueprintMsg{
			path:    inputPath,
			name:    name,
			version: version,
			dir:     dir,
		}
	}
}

type translateBlueprintMsg struct {
	path    string
	name    string
	version string
	dir     string
}

func interpretBlueprint(bp ast.Blueprint) tea.Cmd {
	return func() tea.Msg {
		return interpretBlueprintMsg{
			blueprint: bp,
		}
	}
}

type interpretBlueprintMsg struct {
	blueprint ast.Blueprint
}

func startEvaluateSpec() tea.Cmd {
	return func() tea.Msg {
		return startEvaluateSpecMsg{}
	}
}

type startEvaluateSpecMsg struct {
}

func doneEvaluateSpec() tea.Cmd {
	return func() tea.Msg {
		return doneEvaluateSpecMsg{}
	}
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

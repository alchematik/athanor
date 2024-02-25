package diff

import (
	"context"
	"fmt"
	"sync"

	api "github.com/alchematik/athanor/internal/api/resource"
	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/cli/view/component"
	"github.com/alchematik/athanor/internal/differ"
	"github.com/alchematik/athanor/internal/evaluator"
	consumerpb "github.com/alchematik/athanor/internal/gen/go/proto/blueprint/v1"
	plug "github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/selector"

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

type Show struct {
	Context   context.Context
	Config    Config
	State     string
	InputPath string
	DiffTree  *component.TreeModel
	Spinner   spinner.Model
	Error     error

	Controller *selector.DiffController

	Logger hclog.Logger
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
	Debug   bool
}

func NewShow(params ShowParams) (*tea.Program, error) {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(component.ColorCyan500)
	logger := hclog.NewNullLogger()
	if params.Debug {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			return nil, err
		}
		logger = hclog.New(&hclog.LoggerOptions{
			Output: f,
			Level:  hclog.Debug,
		})
	}
	return tea.NewProgram(&Show{
		Context:   params.Context,
		State:     showStateInitializing,
		InputPath: params.Path,
		DiffTree: &component.TreeModel{
			Spinner: s,
		},
		Logger: logger,
	}), nil
}

func (v *Show) Init() tea.Cmd {
	return tea.Batch(v.Spinner.Tick, loadConfigCmd(v.InputPath))
}

func (v *Show) View() string {
	switch v.State {
	case showStateInitializing:
		return "initializing..."
	case showStateInterpreting:
		return "interpreting..."
	case showStateEvaluating:
		return v.DiffTree.View()
	case showStateError:
		return "ERROR: " + v.Error.Error() + "\n"
	default:
		return ""
	}
}

func (v *Show) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return v, tea.Quit
		}

		return v, nil
	case doneEvaluateSpecMsg:
		return v, tea.Quit
	case configLoadedMsg:
		v.Config = msg.config
		v.State = showStateTranslating
		return v, translateBlueprintCmd(v.Context, v.Config)
	case setSpecMsg:
		v.DiffTree.Root = &component.TreeNode{
			Entries: componentsToEntries(msg.spec.Spec.Components),
		}

		target := evaluator.NewEvaluator(&api.Unresolved{})

		actual := evaluator.NewEvaluator(
			&api.API{
				ProviderPluginManager: plug.NewProvider(v.Config.ProvidersDir, v.Logger),
			},
		)

		d := differ.Differ{
			Lock: &sync.Mutex{},
		}

		v.Controller = selector.NewDiffController(
			v.Logger,
			msg.spec,
			target,
			actual,
			d,
		)

		v.State = showStateEvaluating
		return v, evaluateNext(v.Controller)
	case evaluateNextMsg:
		if len(msg.next) == 0 {
			return v, nil
		}

		var cmds []tea.Cmd
		for _, n := range msg.next {
			cmds = append(cmds, evaluateCmd(v.Logger, v.Context, v.Controller, n))
		}

		return v, tea.Batch(cmds...)
	case setStatusMsg:
		next := v.Controller.Next()
		var cmds []tea.Cmd
		cmds = append(cmds, tea.Batch(
			func() tea.Msg { return evaluateNextMsg{next: next} },
			func() tea.Msg {
				return component.UpdateTreeNodeMsg{
					Selector: msg.selector,
					Status:   component.TreeNodeStatus(msg.status),
				}
			},
		))

		if msg.selector.Parent == nil && msg.status != string(selector.TreeNodeStatusLoading) {
			cmds = append(cmds, func() tea.Msg { return doneEvaluateSpec() })
		}

		return v, tea.Sequence(cmds...)
	case displayErrorMsg:
		v.Error = msg.error
		v.State = showStateError
		return v, quit
	case quitMsg:
		return v, tea.Quit
	default:
		var cmd tea.Cmd
		v.DiffTree, cmd = v.DiffTree.Update(msg)
		return v, cmd
	}
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

package diff

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/internal/ast"
	consumerpb "github.com/alchematik/athanor/internal/gen/go/proto/blueprint/v1"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	showStateInitializing = "initializing"
	showStateTranslating  = "translating"
	showStateInterpreting = "interpreting"
	showStateEvaluating   = "evaluating"
	showStateError        = "error"
)

type Show struct {
	Tree *DiffModel
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
	t := NewDiff(params.Context, params.Path)
	return tea.NewProgram(&Show{
		Tree: t,
	})
}

func (v *Show) Init() tea.Cmd {
	return v.Tree.Init()
}

func (v *Show) View() string {
	return v.Tree.View()
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
	default:
		var cmd tea.Cmd
		v.Tree, cmd = v.Tree.Update(msg)
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

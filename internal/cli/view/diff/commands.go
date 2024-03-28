package diff

import (
	"context"
	"fmt"
	"sort"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/cli/view"
	"github.com/alchematik/athanor/internal/cli/view/component"
	"github.com/alchematik/athanor/internal/dependency"
	"github.com/alchematik/athanor/internal/diff"
	"github.com/alchematik/athanor/internal/interpreter"
	plug "github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/repo"
	"github.com/alchematik/athanor/internal/selector"
	"github.com/alchematik/athanor/internal/spec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hashicorp/go-hclog"
)

type Controller interface {
	Next() []selector.Selector
	Process(context.Context, selector.Selector) (diff.Type, error)
}

func quit() tea.Msg {
	return quitMsg{}
}

type quitMsg struct{}

func evaluateCmd(logger hclog.Logger, ctx context.Context, c Controller, s selector.Selector) tea.Cmd {
	return func() tea.Msg {
		res, err := c.Process(ctx, s)
		if err != nil {
			return view.DisplayError(err)
		}

		return setStatusMsg{
			selector: s,
			diff:     res,
		}
	}
}

type setStatusMsg struct {
	selector selector.Selector
	diff     diff.Type
}

func setStatus(s selector.Selector, status diff.Type) tea.Cmd {
	return func() tea.Msg {
		return setStatusMsg{
			selector: s,
			diff:     status,
		}
	}
}

func interpretBlueprintCmd(ctx context.Context, config view.Config, logger hclog.Logger) tea.Cmd {
	return func() tea.Msg {
		s, err := interpretBlueprint(ctx, config, logger)
		if err != nil {
			return view.DisplayError(err)
		}
		return setSpecMsg{
			spec: s,
		}
	}
}

func interpretBlueprint(ctx context.Context, config view.Config, logger hclog.Logger) (spec.ComponentBuild, error) {
	depManager, err := dependency.NewManager(dependency.ManagerParams{
		LockFilePath: "athanor.lock.json",
	})
	if err != nil {
		return spec.ComponentBuild{}, err
	}

	plugManager := plug.NewPlugManager(logger)
	defer plugManager.Stop()

	s := spec.Spec{
		Components:    map[string]spec.Component{},
		DependencyMap: map[string][]string{},
	}
	var src repo.PluginSource
	switch config.Translator.Repo.Type {
	case "local":
		src = repo.PluginSourceLocal{
			Path: config.Translator.Repo.Path,
		}
	default:
		return spec.ComponentBuild{}, fmt.Errorf("invalid translator repo type: %s", config.Translator.Repo.Type)
	}
	b := ast.StmtBuild{
		Translator: ast.Translator{
			Source: src,
		},
		Build: ast.ExprBuild{
			Alias: config.Name,
			Source: repo.BlueprintSourceFilePath{
				Path: config.InputPath,
			},
			// TODO: fill in.
			Config:        []ast.Expr{},
			RuntimeConfig: ast.ExprNil{},
		},
	}
	in := interpreter.NewInterpreter(plugManager, depManager, s, b)

	next := in.Next()
	for len(next) > 0 {
		for _, n := range next {
			if err := in.Interpret(ctx, n); err != nil {
				return spec.ComponentBuild{}, err
			}
		}

		next = in.Next()
	}

	return spec.ComponentBuild{Spec: s}, nil
}

type setSpecMsg struct {
	spec spec.ComponentBuild
}

func evaluateNext(q Controller) tea.Cmd {
	return func() tea.Msg {
		next := q.Next()
		return evaluateNextMsg{
			next: next,
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

type doneReconcilingMsg struct{}

type sorter []*component.TreeNode

func (s sorter) Len() int {
	return len(s)
}

func (s sorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sorter) Less(i, j int) bool {
	if s[i].Kind == s[j].Kind {
		return s[i].Name < s[j].Name
	}

	if s[i].Kind == "blueprint" {
		return false
	}

	if s[j].Kind == "blueprint" {
		return true
	}

	return s[i].Kind < s[j].Kind
}

func componentsToEntries(components map[string]spec.Component) []*component.TreeNode {
	var out []*component.TreeNode
	for name, comp := range components {
		var sub []*component.TreeNode
		var kind string
		switch comp := comp.(type) {
		case spec.ComponentBuild:
			sub = componentsToEntries(comp.Spec.Components)
			kind = "blueprint"
		case spec.ComponentResource:
			kind = comp.Value.Identifier.ResourceType
		}
		out = append(out, &component.TreeNode{
			Name:    name,
			Kind:    kind,
			Entries: sub,
			Status:  "loading",
		})
	}

	sort.Sort(sorter(out))

	return out
}

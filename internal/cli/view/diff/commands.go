package diff

import (
	"context"
	"encoding/json"
	"os"
	"sort"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/cli/view/component"
	"github.com/alchematik/athanor/internal/interpreter"
	plug "github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/selector"
	"github.com/alchematik/athanor/internal/spec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hashicorp/go-hclog"
)

type Controller interface {
	Next() []selector.Selector
	Process(context.Context, selector.Selector) (selector.TreeNodeStatus, error)
}

func quit() tea.Msg {
	return quitMsg{}
}

type quitMsg struct{}

func evaluateCmd(logger hclog.Logger, ctx context.Context, c Controller, s selector.Selector) tea.Cmd {
	return func() tea.Msg {
		res, err := c.Process(ctx, s)
		if err != nil {
			return displayError(err)
		}

		return setStatusMsg{
			selector: s,
			status:   string(res),
		}
	}
}

type setStatusMsg struct {
	selector selector.Selector
	status   string
}

func setStatus(s selector.Selector, status string) tea.Cmd {
	return func() tea.Msg {
		return setStatusMsg{
			selector: s,
			status:   status,
		}
	}
}

func loadConfigCmd(configPath string) tea.Cmd {
	return func() tea.Msg {
		c, err := loadConfig(configPath)
		if err != nil {
			return displayError(err)
		}

		return configLoadedMsg{config: c}
	}
}

func loadConfig(configPath string) (Config, error) {
	f, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	var c Config
	if err := json.Unmarshal(f, &c); err != nil {
		return Config{}, err
	}

	return c, nil
}

type configLoadedMsg struct {
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

func interpretBlueprintCmd(ctx context.Context, config Config) tea.Cmd {
	return func() tea.Msg {
		s, err := interpretBlueprint(ctx, config)
		if err != nil {
			return displayError(err)
		}
		return setSpecMsg{
			spec: s,
		}
	}
}

func interpretBlueprint(ctx context.Context, config Config) (spec.ComponentBuild, error) {
	in := interpreter.Interpreter{
		Translator: plug.NewTranslator(),
	}
	s := spec.Spec{
		Components:    map[string]spec.Component{},
		DependencyMap: map[string][]string{},
	}
	if err := in.Interpret(ctx, s, ast.StmtBuild{
		Alias: config.Name,
		Repo: ast.RepoLocal{
			Path: config.InputPath,
		},
		Translator: ast.Translator{
			Name:    config.Translator.Name,
			Version: config.Translator.Version,
			Repo: ast.RepoLocal{
				Path: config.TranslatorsDir,
			},
		},
		// TODO: fill in.
		RuntimeConfig: ast.ExprNil{},
	}); err != nil {
		return spec.ComponentBuild{}, err
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
		return true
	}

	if s[j].Kind == "blueprint" {
		return false
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

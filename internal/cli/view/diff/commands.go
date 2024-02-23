package diff

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/cli/view/component"
	"github.com/alchematik/athanor/internal/diff"
	consumerpb "github.com/alchematik/athanor/internal/gen/go/proto/blueprint/v1"
	translatorpb "github.com/alchematik/athanor/internal/gen/go/proto/translator/v1"
	"github.com/alchematik/athanor/internal/interpreter"
	plug "github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/selector"
	"github.com/alchematik/athanor/internal/spec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hashicorp/go-hclog"
)

func quit() tea.Msg {
	return quitMsg{}
}

type quitMsg struct{}

func evaluateCmd(logger hclog.Logger, ctx context.Context, s selector.Selector, differ diff.Differ, q *selector.Queuer) tea.Cmd {
	return func() tea.Msg {
		res, err := evaluate(ctx, s, differ, q)
		if err != nil {
			return displayError(err)
		}

		return setStatusMsg{
			selector: s,
			status:   string(res.Operation()),
		}
	}
}

func evaluate(ctx context.Context, s selector.Selector, differ diff.Differ, q *selector.Queuer) (diff.Type, error) {
	res, err := differ.Diff(ctx, s)
	if err != nil {
		return nil, err
	}

	q.Done(s)

	return res, nil
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

func translateBlueprintCmd(ctx context.Context, config Config) tea.Cmd {
	return func() tea.Msg {
		s, err := translateBlueprint(ctx, config)
		if err != nil {
			return displayError(err)
		}
		return setSpecMsg{
			spec: s,
		}
	}
}

func translateBlueprint(ctx context.Context, config Config) (spec.Spec, error) {
	translatorPlugManager := plug.Translator{
		Dir: config.TranslatorsDir,
	}

	client, stop, err := translatorPlugManager.Client(config.Translator.Name, config.Translator.Version)
	if err != nil {
		return spec.Spec{}, err
	}
	defer stop()

	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		return spec.Spec{}, err
	}

	defer os.Remove(tempFile.Name())

	_, err = client.TranslateBlueprint(ctx, &translatorpb.TranslateBlueprintRequest{
		InputPath:  config.InputPath,
		OutputPath: tempFile.Name(),
	})
	if err != nil {
		return spec.Spec{}, fmt.Errorf("error translating blueprint: %v", err)
	}

	blueprintData, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return spec.Spec{}, err
	}

	var blueprint consumerpb.Blueprint
	if err := json.Unmarshal(blueprintData, &blueprint); err != nil {
		return spec.Spec{}, fmt.Errorf("error unmarshaling blueprint: %v", err)
	}

	bp, err := convertBlueprint(&blueprint)
	if err != nil {
		return spec.Spec{}, fmt.Errorf("error converting blueprint: %v", err)
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
		return spec.Spec{}, err
	}

	return s, nil

}

type setSpecMsg struct {
	spec spec.Spec
}

func evaluateNext(q *selector.Queuer) tea.Cmd {
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

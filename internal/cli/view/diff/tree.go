package diff

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"

	api "github.com/alchematik/athanor/internal/api/resource"
	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/cli/view/component"
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

type DiffModel struct {
	Context         context.Context
	InputPath       string
	State           string
	Spec            spec.Spec
	TargetEvaluator *evaluator.Evaluator
	ActualEvaluator *evaluator.Evaluator
	Differ          diff.Differ
	Config          Config
	Queuer          *selector.Queuer
	Tree            *component.TreeModel
	Error           error
}

func NewDiff(ctx context.Context, inputPath string) *DiffModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(component.ColorCyan500)
	return &DiffModel{
		Context:   ctx,
		State:     showStateInitializing,
		InputPath: inputPath,
		Tree: &component.TreeModel{
			Spinner: s,
		},
	}
}

func (t *DiffModel) Init() tea.Cmd {
	return tea.Batch(t.Tree.Init(), loadConfigCmd(t.InputPath))
}

func (t *DiffModel) View() string {
	switch t.State {
	case showStateInitializing:
		return "initializing..."
	case showStateInterpreting:
		return "interpreting..."
	case showStateEvaluating:
		return t.Tree.View()
	case showStateError:
		return "ERROR: " + t.Error.Error()
	default:
		return ""
	}
}

func (t *DiffModel) Update(msg tea.Msg) (*DiffModel, tea.Cmd) {
	switch msg := msg.(type) {
	case configLoadedMsg:
		t.Config = msg.config
		t.State = showStateTranslating
		return t, translateBlueprintCmd(t.Context, t.Config)
	case setSpecMsg:
		t.Spec = msg.spec
		entries := componentsToEntries(msg.spec.Components)
		t.Tree.Root = &component.TreeNode{
			Entries: entries,
		}

		q := selector.NewQueuer(t.Config.Name, msg.spec)
		t.Queuer = q

		t.TargetEvaluator = evaluator.NewEvaluator(
			&api.Unresolved{},
			t.Spec,
			state.Environment{
				States:        map[string]state.Type{},
				DependencyMap: map[string][]string{},
			},
		)

		p := plug.NewProvider()
		p.Dir = t.Config.ProvidersDir
		p.Logger = hclog.NewNullLogger()
		t.ActualEvaluator = evaluator.NewEvaluator(
			&api.API{
				ProviderPluginManager: p,
			},
			t.Spec,
			state.Environment{
				States:        map[string]state.Type{},
				DependencyMap: map[string][]string{},
			},
		)
		t.Differ = diff.Differ{
			Target: t.TargetEvaluator.Env,
			Actual: t.ActualEvaluator.Env,
			Result: diff.Environment{
				Diffs: map[string]diff.Type{},
			},
			Lock: &sync.Mutex{},
		}
		t.State = showStateEvaluating
		return t, evaluateNext(t.Queuer)
	case evaluateNextMsg:
		if len(msg.next) == 0 {
			return t, nil
		}

		var cmds []tea.Cmd
		for _, n := range msg.next {
			cmds = append(cmds, evaluateCmd(t.Context, n, t.TargetEvaluator, t.ActualEvaluator, t.Differ, t.Queuer))
		}
		return t, tea.Batch(cmds...)
	case doneEvaluateSpecMsg:
		return t, nil
	case setStatusMsg:
		next := t.Queuer.Next()
		var cmds []tea.Cmd
		cmds = append(cmds, tea.Batch(
			func() tea.Msg { return evaluateNextMsg{next: next} },
			func() tea.Msg {
				// Initial evaluation of a blueprint is empty. Diff status is set later.
				st := msg.status
				if st == "" {
					st = "loading"
				}
				return component.UpdateTreeNodeMsg{
					Selector: msg.selector,
					Status:   component.TreeNodeStatus(st),
				}
			},
		))

		if msg.selector.Parent == nil {
			if t.Differ.Result.Diffs[msg.selector.Name].Operation() != diff.OperationEmpty {
				cmds = append(cmds, func() tea.Msg { return doneEvaluateSpec() })
			}
		}

		return t, tea.Sequence(cmds...)

	case displayErrorMsg:
		t.Error = msg.error
		t.State = showStateError
		return t, tea.Quit
	default:
		var cmd tea.Cmd
		t.Tree, cmd = t.Tree.Update(msg)
		return t, cmd
	}
}

func evaluateCmd(ctx context.Context, s selector.Selector, target, actual *evaluator.Evaluator, differ diff.Differ, q *selector.Queuer) tea.Cmd {
	return func() tea.Msg {
		res, err := evaluate(ctx, s, target, actual, differ, q)
		if err != nil {
			return displayError(err)
		}

		return setStatusMsg{
			selector: s,
			status:   string(res.Operation()),
		}
	}
}

func evaluate(ctx context.Context, s selector.Selector, target, actual *evaluator.Evaluator, differ diff.Differ, q *selector.Queuer) (diff.Type, error) {
	if err := target.Eval(ctx, s); err != nil {
		return nil, err
	}

	if err := actual.Eval(ctx, s); err != nil {
		return nil, err
	}

	res, err := differ.Diff(s)
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

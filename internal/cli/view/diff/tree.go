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

type LoaderTree struct {
	Spec    spec.Spec
	Spinner spinner.Model
	Queuer  selector.Queuer
}

func (t LoaderTree) Init() tea.Cmd {
	return func() tea.Msg {
		var cmds []tea.Cmd
		return tea.Batch(cmds...)
	}
}

// func (t LoaderTree) View() string {
// 	rows := t.loaderTreeRow(0, t.Diff, t.Spec, t.Diff.Result)
// 	str := ""
// 	for _, r := range rows {
// 		str += fmt.Sprintf("%s %s\n", r[0], r[1])
// 	}
// 	return str
// }

func (t LoaderTree) loaderTreeRow(spaces int, differ diff.Differ, s spec.Spec, envDiff diff.Environment) [][]string {
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
			st := t.Spinner.View()
			differ.Lock.Lock()
			if d, ok := envDiff.Diffs[r]; ok {
				st = sign(d.Operation())
			}
			differ.Lock.Unlock()
			tree := "├─"
			if i == len(resourceTypes)-1 && j == len(resources)-1 && len(builds) == 0 {
				tree = "└─"
			}
			out = append(out, []string{st, strings.Repeat(" ", spaces) + tree + " " + rt + "/" + r})
		}
	}

	for i, b := range builds {
		build := s.Components[b].(spec.ComponentBuild)

		st := t.Spinner.View()
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
		sub := t.loaderTreeRow(spaces+1, differ, build.Spec, subEnv)
		out = append(out, sub...)
	}

	return out
}

func (t LoaderTree) Update(msg tea.Msg) (LoaderTree, tea.Cmd) {
	return t, nil
}

type Tree struct {
	Context         context.Context
	InputPath       string
	State           string
	Spec            spec.Spec
	TargetEvaluator *evaluator.Evaluator
	ActualEvaluator *evaluator.Evaluator
	Diff            diff.Differ
	Config          Config
	Spinner         spinner.Model
	Queuer          *selector.Queuer
	Error           error
}

func NewTree(ctx context.Context, inputPath string) *Tree {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(ColorCyan500)
	return &Tree{
		Spinner:   s,
		Context:   ctx,
		State:     showStateInitializing,
		InputPath: inputPath,
	}
}

func (t *Tree) Init() tea.Cmd {
	return tea.Batch(t.Spinner.Tick, loadConfig(t.InputPath))
}

func (t *Tree) View() string {
	switch t.State {
	case showStateInitializing:
		return "initializing..."
	case showStateInterpreting:
		return "interpreting..."
	case showStateEvaluating:
		rows := rows(0, t.Spinner, t.Diff, t.Spec, t.Diff.Result)
		str := ""
		for _, r := range rows {
			str += fmt.Sprintf("%s %s\n", r[0], r[1])
		}
		return str
	case showStateError:
		return "ERROR: " + t.Error.Error()
	default:
		return ""
	}
}

func (t *Tree) Update(msg tea.Msg) (*Tree, tea.Cmd) {
	switch msg := msg.(type) {
	case setConfigMsg:
		t.Config = msg.config
		t.State = showStateTranslating
		return t, translateBlueprint(
			t.Context,
			t.Config,
		)
	case setSpecMsg:
		t.Spec = msg.spec

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
		t.Diff = diff.Differ{
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
			cmds = append(cmds, evaluate(t.Context, t.Queuer, n, t.TargetEvaluator, t.ActualEvaluator, t.Diff))
		}
		return t, tea.Batch(cmds...)
	case doneEvaluateSpecMsg:
		return t, nil
	case displayErrorMsg:
		t.Error = msg.error
		t.State = showStateError
		return t, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		t.Spinner, cmd = t.Spinner.Update(msg)
		return t, cmd
	default:
		return t, nil
	}
}

func rows(spaces int, spin spinner.Model, differ diff.Differ, s spec.Spec, envDiff diff.Environment) [][]string {
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
			st := spin.View()
			differ.Lock.Lock()
			if d, ok := envDiff.Diffs[r]; ok {
				st = sign(d.Operation())
			}
			differ.Lock.Unlock()
			tree := "├─"
			if i == len(resourceTypes)-1 && j == len(resources)-1 && len(builds) == 0 {
				tree = "└─"
			}
			out = append(out, []string{st, strings.Repeat(" ", spaces) + tree + " " + rt + "/" + r})
		}
	}

	for i, b := range builds {
		build := s.Components[b].(spec.ComponentBuild)

		st := spin.View()
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
		sub := rows(spaces+1, spin, differ, build.Spec, subEnv)
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

func evaluate(ctx context.Context, q *selector.Queuer, s selector.Selector, target, actual *evaluator.Evaluator, d diff.Differ) tea.Cmd {
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

		q.Done(s)

		if s.Parent == nil {
			if d.Result.Diffs[s.Name].Operation() != diff.OperationEmpty {
				return doneEvaluateSpec()
			}
		}

		return evaluateNextMsg{next: q.Next()}
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
	panic(err)
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

func evaluateNext(q *selector.Queuer) tea.Cmd {
	return func() tea.Msg {
		next := q.Next()
		fmt.Printf("NEXT> %v\n", next)
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

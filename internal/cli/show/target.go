package show

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	external_ast "github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/internal/convert"
	"github.com/alchematik/athanor/internal/dag"
	"github.com/alchematik/athanor/internal/eval"
	"github.com/alchematik/athanor/internal/scope"
	"github.com/alchematik/athanor/internal/state"

	"github.com/bytecodealliance/wasmtime-go/v20"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v3"
)

func NewShowTargetCommand() *cli.Command {
	return &cli.Command{
		Name: "target",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-file",
				Usage: "path to file to write logs to",
			},
			&cli.StringFlag{
				Name:  "config",
				Usage: "path to config file",
			},
		},
		Action: TargetAction,
	}
}

func TargetAction(ctx context.Context, cmd *cli.Command) error {
	inputPath := cmd.Args().First()
	logFilePath := cmd.String("log-file")
	configFilePath := cmd.String("config")

	initState := &Init{
		inputPath:  inputPath,
		configPath: configFilePath,
		context:    ctx,
	}
	if logFilePath != "" {
		f, err := tea.LogToFile(logFilePath, "")
		if err != nil {
			return err
		}

		initState.logger = slog.New(slog.NewTextHandler(f, nil))
	}
	_, err := tea.NewProgram(&TargetModel{current: initState}).Run()
	return err
}

type TargetModel struct {
	current tea.Model
}

func (m *TargetModel) Init() tea.Cmd {
	return m.current.Init()
}

func (m *TargetModel) View() string {
	return m.current.View()
}

func (m *TargetModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.current.Update(msg)
	m.current = next
	return m, cmd
}

type Quit struct {
	logger *slog.Logger
}

func (s *Quit) Init() tea.Cmd {
	return func() tea.Msg {
		return "quit"
	}
}

func (s *Quit) View() string {
	return "quitting..."
}

func (s *Quit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return s, tea.Quit
}

type ErrorModel struct {
	logger *slog.Logger
	error  error
}

func (e *ErrorModel) Init() tea.Cmd {
	return tea.Printf("error: %s", e.error)
}

func (e *ErrorModel) View() string {
	return e.error.Error()
}

func (e *ErrorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return e, tea.Quit
}

type Init struct {
	logger     *slog.Logger
	inputPath  string
	configPath string
	scope      *scope.Scope
	state      *state.State
	context    context.Context
}

type interpreter struct {
	logger *slog.Logger
}

func (it *interpreter) InterpretBlueprint(source external_ast.BlueprintSource, input map[string]any) (external_ast.Blueprint, error) {
	engine := wasmtime.NewEngine()
	module, err := wasmtime.NewModuleFromFile(engine, source.LocalFile.Path)
	if err != nil {
		return external_ast.Blueprint{}, err
	}

	linker := wasmtime.NewLinker(engine)
	if err := linker.DefineWasi(); err != nil {
		return external_ast.Blueprint{}, err
	}

	wasiConfig := wasmtime.NewWasiConfig()

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		return external_ast.Blueprint{}, err
	}
	defer os.RemoveAll(dir)

	if err := wasiConfig.PreopenDir(dir, "/"); err != nil {
		return external_ast.Blueprint{}, err
	}

	store := wasmtime.NewStore(engine)
	store.SetWasi(wasiConfig)

	instance, err := linker.Instantiate(store, module)
	if err != nil {
		return external_ast.Blueprint{}, err
	}

	nom := instance.GetFunc(store, "_start")
	_, err = nom.Call(store)
	if err != nil {
		var wasmtimeError *wasmtime.Error
		if errors.As(err, &wasmtimeError) {
			st, ok := wasmtimeError.ExitStatus()
			if ok && st != 0 {
				return external_ast.Blueprint{}, fmt.Errorf("non-0 exit status: %d", st)
			}
		} else {
			return external_ast.Blueprint{}, err
		}
	}

	data, err := os.ReadFile(filepath.Join(dir, "blueprint.json"))
	if err != nil {
		return external_ast.Blueprint{}, err
	}

	var bp external_ast.Blueprint
	if err := json.Unmarshal(data, &bp); err != nil {
		return external_ast.Blueprint{}, err
	}

	return bp, nil
}

func (s *Init) Init() tea.Cmd {
	s.scope = scope.NewScope()
	s.state = &state.State{
		Resources: map[string]*state.ResourceState{},
		Builds:    map[string]*state.BuildState{},
	}

	return func() tea.Msg {
		c := convert.Converter{
			Logger:               s.logger,
			BlueprintInterpreter: &interpreter{logger: s.logger},
		}
		b := external_ast.DeclareBuild{
			Name: "Build",
			Runtimeinput: external_ast.Expr{
				Value: external_ast.MapCollection{
					Value: map[string]external_ast.Expr{},
				},
			},
			BlueprintSource: external_ast.BlueprintSource{
				LocalFile: external_ast.BlueprintSourceLocalFile{
					Path: s.inputPath,
				},
			},
		}
		if _, err := c.ConvertBuildStmt(s.state, s.scope, b); err != nil {
			return errorMsg{error: err}
		}

		return "done"
	}
}

func (s *Init) View() string {
	return render(0, s.state, s.scope.Build())
}

func render(space int, s *state.State, build *scope.Build) string {
	var out string
	for _, id := range build.Resources() {
		rs, ok := s.ResourceState(id)
		if !ok {
			panic("resource not in state: " + id)
		}

		r := rs.GetResource()
		status := rs.GetEvalState()
		action := rs.GetComponentAction()

		out += ">>" + status.State + " " + string(action) + " " + strings.Repeat(" ", space) + r.Name + "\n"
	}
	for _, id := range build.Builds() {
		bs, ok := s.BuildState(id)
		if !ok {
			panic("build not in state: " + id)
		}

		b := bs.GetBuild()
		status := bs.GetEvalState()

		out += ">>" + status.State + " " + strings.Repeat(" ", space) + b.Name + "\n"
		sub := build.Build(id)
		out += render(space+2, s, sub)
	}

	return out
}

func (s *Init) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			next := &Quit{logger: s.logger}
			return next, next.Init()
		}

		return s, nil
	case errorMsg:
		next := &ErrorModel{logger: s.logger, error: msg.error}
		return next, next.Init()
	case string:
		iter := s.scope.NewIterator()
		next := &EvalModel{
			logger:    s.logger,
			state:     state.NewGlobal(s.state, nil),
			iter:      iter,
			evaluator: eval.NewTargetEvaluator(iter),
			scope:     s.scope,
			context:   s.context,
		}
		next.evaluator.Logger = s.logger

		return next, next.Init()
	default:
		return s, nil
	}
}

type EvalModel struct {
	evaluator *eval.TargetEvaluator
	state     *state.Global
	logger    *slog.Logger
	scope     *scope.Scope
	iter      *dag.Iterator
	context   context.Context
}

func (m *EvalModel) Init() tea.Cmd {
	ids := m.evaluator.Next()
	cmds := make([]tea.Cmd, len(ids))
	for i, id := range ids {
		cmds[i] = func() tea.Msg { return evalMsg{id: id} }
	}
	return tea.Batch(cmds...)
}

func (m *EvalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			next := &Quit{logger: m.logger}
			return next, next.Init()
		}

		return m, nil
	case errorMsg:
		next := &ErrorModel{logger: m.logger, error: msg.error}
		return next, next.Init()
	case evalMsg:
		return m, func() tea.Msg {
			comp, ok := m.scope.Component(msg.id)
			if !ok {
				return errorMsg{error: fmt.Errorf("component not found: %s", msg.id)}
			}

			err := m.evaluator.Eval(m.context, m.state, comp)
			if err != nil {
				return errorMsg{error: err}
			}

			next := m.evaluator.Next()

			return nextMsg{next: next}
		}
	case nextMsg:
		if len(msg.next) == 0 {
			return m, func() tea.Msg { return "done" }
		}
		cmds := make([]tea.Cmd, len(msg.next))
		for i, id := range msg.next {
			cmds[i] = func() tea.Msg { return evalMsg{id: id} }
		}

		return m, tea.Batch(cmds...)
	case string:
		m.logger.Info("done")
		return m, nil
	default:
		return m, nil
	}
}

func (m *EvalModel) View() string {
	return render(0, m.state.Target(), m.scope.Build())
}

type evalMsg struct {
	id string
}

type nextMsg struct {
	next []string
}

type errorMsg struct {
	error error
}

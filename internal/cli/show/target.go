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
	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/convert"

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
	scope      *ast.Scope
	global     *ast.Global
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
	s.scope = ast.NewScope("")
	s.global = ast.NewGlobal()

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
		if _, err := c.ConvertBuildStmt(s.global, s.scope, b); err != nil {
			return errorMsg{error: err}
		}

		return "done"
	}
}

func (s *Init) View() string {
	return render(0, s.scope)
}

func render(space int, scope *ast.Scope) string {
	var out string
	for _, name := range scope.Resources() {
		out += strings.Repeat(" ", space) + name + "\n"
	}
	for _, b := range scope.Builds() {
		out += strings.Repeat(" ", space) + b.Name() + "\n"
		out += render(space+2, b)
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
		// if msg == "done" {
		// 	next := &Quit{logger: s.logger}
		// 	return next, next.Init()
		// }

		return s, nil
	default:
		return s, nil
	}
}

type errorMsg struct {
	error error
}

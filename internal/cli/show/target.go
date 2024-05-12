package show

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	// "os"

	external_ast "github.com/alchematik/athanor/ast"
	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/dag"

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
		return quitMsg{}
	}
}

func (s *Quit) View() string {
	return "quitting..."
}

func (s *Quit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return s, tea.Quit
}

/*

- Evaluate top level build node
	- Read input WASI file
	- Red config file if specified
	- Run wasi file
		- Will output file
- Read output file
- EvaluateAST
	- Build up dependency graph.
	- Produce environment
- Display environment

*/

type Init struct {
	logger     *slog.Logger
	inputPath  string
	configPath string
	context    ast.Context
}

type interpreter struct {
	logger *slog.Logger
}

func (it *interpreter) InterpretBlueprint(source external_ast.BlueprintSource, input map[string]any) (external_ast.Blueprint, error) {
	it.logger.Info("INTERPRETING BLUEPRINT", "file", source.LocalFile.Path)
	// data, err := os.ReadFile(source.LocalFile.Path)
	// if err != nil {
	// 	return ast.Blueprint{}, err
	// }

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

	it.logger.Info("TEMP DIR >>>>>", "dir", dir)

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
		it.logger.Info("GOT ERROR >>>", "err", err)
		if errors.As(err, &wasmtimeError) {
			st, ok := wasmtimeError.ExitStatus()
			it.logger.Info("WASM ERROR >>>", "code", st, "ok", ok)
			if ok && st != 0 {
				return external_ast.Blueprint{}, fmt.Errorf("non-0 exit status: %d", st)
			}
		} else {
			return external_ast.Blueprint{}, err
		}
	}

	it.logger.Info("GUESS IT WAS NOTHING, reading file", "path", filepath.Join(dir, "blueprint.json"))

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
	s.context = ast.NewContext()
	f := func() tea.Msg {
		c := ast.Converter{
			Logger:               s.logger,
			BlueprintInterpreter: &interpreter{logger: s.logger},
		}
		b := external_ast.DeclareBuild{
			Name: "",
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
		_, err := c.ConvertBuildStmt(s.context, b)
		s.logger.Info("done converting", "error", err)
		return "done"
	}
	return f
}

func (s *Init) View() string {
	g := s.context.DAG()
	iter := dag.InitIterator(g)
	var out string
	nodes := iter.Next()
	for len(nodes) > 0 {
		s.logger.Info("nodes!!", "nodes", nodes)
		for _, n := range nodes {
			if iter.Visited(n) {
				iter.Done(n)
				continue
			}

			iter.Start(n)
			out += n + "\n"
		}

		nodes = iter.Next()
	}

	return out + "!"
}

func (s *Init) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			next := &Quit{logger: s.logger}
			return next, next.Init()
		}

		return s, nil
	case string:
		s.logger.Info("done converting!!", "graph", s.context.DAG())
		// if msg == "done" {
		// 	next := &Quit{logger: s.logger}
		// 	return next, next.Init()
		// }

		return s, nil
	default:
		return s, nil
	}
}

type quitMsg struct{}

package deps

import (
	"context"
	"fmt"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/cli/view"
	"github.com/alchematik/athanor/internal/dependency"
	"github.com/alchematik/athanor/internal/interpreter"
	plug "github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/repo"
	"github.com/alchematik/athanor/internal/spec"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hashicorp/go-hclog"
)

type Install struct {
	Context   context.Context
	Logger    hclog.Logger
	InputPath string
	Config    view.Config
	Error     error
	Upgrade   bool
}

type InstallParams struct {
	Context context.Context
	Path    string
	Debug   bool
	Upgrade bool
}

func NewInstall(params InstallParams) (*tea.Program, error) {
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
	return tea.NewProgram(&Install{
		Context:   params.Context,
		Logger:    logger,
		InputPath: params.Path,
		Upgrade:   params.Upgrade,
	}), nil
}

func (m *Install) Init() tea.Cmd {
	return view.LoadConfigCmd(m.InputPath)
}

func (m *Install) View() string {
	if m.Error != nil {
		return m.Error.Error()
	}

	return "install!"
}

func (m *Install) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}

		return m, nil
	case view.DisplayErrorMsg:
		m.Error = msg.Error
		return m, nil
	case view.ConfigLoadedMsg:
		m.Config = msg.Config
		return m, func() tea.Msg {
			depManager, err := dependency.NewManager(dependency.ManagerParams{
				LockFilePath: "athanor.lock.json",
				FetchRemote:  true,
				Upgrade:      m.Upgrade,
			})
			if err != nil {
				return view.DisplayError(err)
			}

			plugManager := plug.NewPlugManager(m.Logger)
			defer plugManager.Stop()

			in := interpreter.Interpreter{
				DepManager:  depManager,
				PlugManager: plugManager,
			}
			s := spec.Spec{
				Components:    map[string]spec.Component{},
				DependencyMap: map[string][]string{},
			}
			var src repo.Source
			switch m.Config.Translator.Repo.Type {
			case "local":
				src = repo.Local{
					Path: m.Config.Translator.Repo.Path,
				}
			default:
				return view.DisplayError(fmt.Errorf("invalid translator repo type: %s", m.Config.Translator.Repo.Type))
			}

			if err := in.Interpret(m.Context, s, ast.StmtBuild{
				Translator: ast.Translator{
					Source: src,
				},
				Build: ast.ExprBuild{
					Alias: m.Config.Name,
					Source: repo.Local{
						Path: m.Config.InputPath,
					},
					// TODO: fill in.
					Config:        []ast.Expr{},
					RuntimeConfig: ast.ExprNil{},
				},
			}); err != nil {
				return view.DisplayError(fmt.Errorf("error interpreting: %s", err))
			}

			if err := depManager.FlushLockFile(); err != nil {
				return view.DisplayError(err)
			}

			return nil
		}
	default:
		return m, nil
	}
}

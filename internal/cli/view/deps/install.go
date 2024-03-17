package deps

import (
	"context"
	"fmt"
	"strings"

	"github.com/alchematik/athanor/internal/ast"
	"github.com/alchematik/athanor/internal/cli/view"
	"github.com/alchematik/athanor/internal/cli/view/component"
	"github.com/alchematik/athanor/internal/dependency"
	"github.com/alchematik/athanor/internal/interpreter"
	plug "github.com/alchematik/athanor/internal/plugin"
	"github.com/alchematik/athanor/internal/repo"
	"github.com/alchematik/athanor/internal/spec"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hashicorp/go-hclog"
)

type Install struct {
	Context   context.Context
	Logger    hclog.Logger
	InputPath string
	Config    view.Config
	Error     error
	Upgrade   bool

	downloads []string
	status    map[string]string
	spinner   spinner.Model
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

	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(component.ColorCyan500)

	return tea.NewProgram(&Install{
		Context:   params.Context,
		Logger:    logger,
		InputPath: params.Path,
		Upgrade:   params.Upgrade,
		status:    map[string]string{},
		spinner:   s,
	}), nil
}

func (m *Install) Init() tea.Cmd {
	return view.LoadConfigCmd(m.InputPath)
}

func (m *Install) View() string {
	if m.Error != nil {
		return m.Error.Error()
	}

	entries := make([]string, len(m.downloads))
	for i, entry := range m.downloads {
		status := m.status[entry]
		var statusDisplay string
		switch status {
		case "downloading":
			statusDisplay = m.spinner.View()
		case "done":
			// TODO: Extract component.
			statusDisplay = lipgloss.NewStyle().Foreground(component.ColorGreen400).Render("✓")
		}

		entries[i] = fmt.Sprintf("%s %s", statusDisplay, entry)
	}

	return strings.Join(entries, "\n")
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
				OnDownloadStart: func(s string, src any) {
					var entry string
					switch src := src.(type) {
					case dependency.SourceGitHubRelease:
						entry = fmt.Sprintf("github.com/%s/%s@%s", src.RepoOwner, src.RepoName, src.Name)
					}

					if _, ok := m.status[entry]; ok {
						return
					}

					// TODO: Make this thread safe.
					m.downloads = append(m.downloads, entry)
					m.status[entry] = "downloading"
				},
				OnDownloadSuccess: func(s string, src any) {
					var entry string
					switch src := src.(type) {
					case dependency.SourceGitHubRelease:
						entry = fmt.Sprintf("github.com/%s/%s@%s", src.RepoOwner, src.RepoName, src.Name)
					}

					// TODO: Make this thread safe.
					m.downloads = append(m.downloads, entry)
					m.status[entry] = "done"
				},
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

			return doneMsg{}
		}
	case doneMsg:
		return m, tea.Quit
	default:
		s, spinMsg := m.spinner.Update(msg)
		m.spinner = s
		return m, spinMsg
	}
}

type doneMsg struct {
}

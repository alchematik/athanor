package deps

import (
	"context"
	"fmt"
	"runtime"
	"sync"

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

	downloads   []string
	status      map[string]string
	spinner     spinner.Model
	interpreter *interpreter.Interpreter
	depManager  *dependency.Manager
	plugManager *plug.Manager
	lock        sync.Mutex
	inflight    int
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
	return tea.Batch(view.LoadConfigCmd(m.InputPath), m.spinner.Tick)
}

func (m *Install) View() string {
	if m.Error != nil {
		return m.Error.Error()
	}

	var entries string
	for _, entry := range m.downloads {
		status := m.status[entry]
		var statusDisplay string
		switch status {
		case "downloading":
			statusDisplay = m.spinner.View()
		case "done":
			// TODO: Extract component.
			statusDisplay = lipgloss.NewStyle().Foreground(component.ColorGreen400).Render("✓")
		}

		entries += fmt.Sprintf("%s %s\n", statusDisplay, entry)
	}

	return entries
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

		depManager, err := dependency.NewManager(dependency.ManagerParams{
			LockFilePath: "athanor.lock.json",
			FetchRemote:  true,
			Upgrade:      m.Upgrade,
			// OnDownloadStart: func(s string, src any) {
			// 	var entry string
			// 	switch src := src.(type) {
			// 	case dependency.SourceGitHubRelease:
			// 		entry = fmt.Sprintf("github.com/%s/%s@%s", src.RepoOwner, src.RepoName, src.Name)
			// 	}
			//
			// 	if _, ok := m.status[entry]; ok {
			// 		return
			// 	}
			//
			// 	// TODO: Make this thread safe.
			// 	m.downloads = append(m.downloads, entry)
			// 	m.status[entry] = "downloading"
			// },
			// OnDownloadSuccess: func(s string, src any) {
			// 	var entry string
			// 	switch src := src.(type) {
			// 	case dependency.SourceGitHubRelease:
			// 		entry = fmt.Sprintf("github.com/%s/%s@%s", src.RepoOwner, src.RepoName, src.Name)
			// 	}
			//
			// 	// TODO: Make this thread safe.
			// 	m.downloads = append(m.downloads, entry)
			// 	m.status[entry] = "done"
			// },
		})
		if err != nil {
			return m, func() tea.Msg { return view.DisplayError(err) }
		}

		m.depManager = depManager

		m.plugManager = plug.NewPlugManager(m.Logger)

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
			return m, func() tea.Msg {
				return view.DisplayError(fmt.Errorf("invalid translator repo type: %s", m.Config.Translator.Repo.Type))
			}
		}

		b := ast.StmtBuild{
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
		}

		m.interpreter = interpreter.NewInterpreter(m.plugManager, depManager, s, b)
		m.inflight = 1

		return m, func() tea.Msg { return nextMsg{} }
	case nextMsg:
		next := m.interpreter.Next()

		m.lock.Lock()
		m.inflight--
		m.inflight += len(next)
		m.lock.Unlock()

		if m.inflight == 0 {
			return m, func() tea.Msg { return doneMsg{} }
		}

		var batch []tea.Cmd
		for _, n := range next {
			n := n
			var entry string
			var dep dependency.BinDependency
			switch stmt := n.Stmt.(type) {
			case ast.StmtBuild:
				switch src := stmt.Translator.Source.(type) {
				case repo.GitHubRelease:
					entry = fmt.Sprintf("github.com/%s/%s@%s", src.RepoOwner, src.RepoName, src.Name)
					dep = dependency.BinDependency{
						Type: "translator",
						Source: dependency.SourceGitHubRelease{
							RepoOwner: src.RepoOwner,
							RepoName:  src.RepoName,
							Name:      src.Name,
						},
						OS:   runtime.GOOS,
						Arch: runtime.GOARCH,
					}
				}
			case ast.StmtResource:
				switch src := stmt.Provider.Source.(type) {
				case repo.GitHubRelease:
					entry = fmt.Sprintf("github.com/%s/%s@%s", src.RepoOwner, src.RepoName, src.Name)
					dep = dependency.BinDependency{
						Type: "provider",
						Source: dependency.SourceGitHubRelease{
							RepoOwner: src.RepoOwner,
							RepoName:  src.RepoName,
							Name:      src.Name,
						},
						OS:   runtime.GOOS,
						Arch: runtime.GOARCH,
					}
				}
			}

			if _, ok := m.status[entry]; !ok && entry != "" {
				isInstalled, err := m.depManager.IsBinDependencyInstalled(dep)
				if err != nil {
					return m, func() tea.Msg { return view.DisplayError(err) }
				}

				if !isInstalled {
					// TODO: is this thread safe?
					m.downloads = append(m.downloads, entry)
					m.status[entry] = "downloading"
				}
			}

			batch = append(batch, func() tea.Msg {
				if err := m.interpreter.Interpret(m.Context, n); err != nil {
					return view.DisplayError(err)
				}

				m.lock.Lock()
				if _, ok := m.status[entry]; ok {
					m.status[entry] = "done"
				}
				m.lock.Unlock()

				return nextMsg{}
			})
		}

		// if len(batch) == 0 {
		// 	return m, func() tea.Msg { return doneMsg{} }
		// }

		return m, tea.Batch(batch...)
	case doneMsg:
		// TODO: Need to make sure this stops even if process errors.
		m.plugManager.Stop()

		if err := m.depManager.FlushLockFile(); err != nil {
			return m, func() tea.Msg { return view.DisplayError(err) }
		}
		return m, tea.Quit
	default:
		s, spinMsg := m.spinner.Update(msg)
		m.spinner = s
		return m, spinMsg
	}
}

type doneMsg struct {
}

type nextMsg struct {
	stmt interpreter.Stmt
}

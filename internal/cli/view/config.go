package view

import (
	"encoding/json"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type Config struct {
	Name       string `json:"name"`
	InputPath  string `json:"input_path"`
	Translator struct {
		Name    string         `json:"name"`
		Version string         `json:"version"`
		Repo    TranslatorRepo `json:"repo"`
	} `json:"translator"`
}

type TranslatorRepo struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

func LoadConfigCmd(configPath string) tea.Cmd {
	return func() tea.Msg {
		c, err := loadConfig(configPath)
		if err != nil {
			return DisplayError(err)
		}

		return ConfigLoadedMsg{Config: c}
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

type ConfigLoadedMsg struct {
	Config Config
}

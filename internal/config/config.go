package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Config is the global config stored at ~/.artifact/config.json.
// Holds credentials and global settings only.
type Config struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url,omitempty"`

	// Deprecated: workspace binding moved to project config.
	// Kept for one-time migration on startup.
	LegacyDefaultWorkspaceID string `json:"default_workspace_id,omitempty"`
}

// ProjectConfig is stored at <cwd>/.artifact/config.json.
// Binds this directory to a workspace.
type ProjectConfig struct {
	WorkspaceID   string    `json:"workspace_id"`
	WorkspaceName string    `json:"workspace_name,omitempty"`
	InitializedAt time.Time `json:"initialized_at"`
}

// ── Global config ─────────────────────────────────────────────────────────────

func globalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".artifact", "config.json"), nil
}

func Load() (*Config, error) {
	path, err := globalConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	path, err := globalConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ── Project config ────────────────────────────────────────────────────────────

func projectConfigPath() string {
	return filepath.Join(".artifact", "config.json")
}

// ProjectInitialized reports whether .artifact/ exists in the cwd.
func ProjectInitialized() bool {
	_, err := os.Stat(".artifact")
	return err == nil
}

func LoadProject() (*ProjectConfig, error) {
	data, err := os.ReadFile(projectConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &ProjectConfig{}, nil
		}
		return nil, err
	}
	var pc ProjectConfig
	if err := json.Unmarshal(data, &pc); err != nil {
		return nil, err
	}
	return &pc, nil
}

func SaveProject(pc *ProjectConfig) error {
	if err := os.MkdirAll(filepath.Dir(projectConfigPath()), 0755); err != nil {
		return err
	}
	if pc.InitializedAt.IsZero() {
		pc.InitializedAt = time.Now()
	}
	data, err := json.MarshalIndent(pc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(projectConfigPath(), data, 0600)
}

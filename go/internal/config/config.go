// Package config loads the small JSON configuration file and chooses sensible
// XDG paths for the shared data store.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Config is shared by the Go and Python versions.
type Config struct {
	Root    string              `json:"root"`
	Ignore  []string            `json:"ignore,omitempty"`
	Openers map[string][]string `json:"openers,omitempty"`
}

var defaultIgnore = []string{
	"go", "python", "bin", "dist", "node_modules", ".git", ".venv", "venv",
}

func Default(root string) Config {
	root = Expand(root)
	abs, err := filepath.Abs(root)
	if err == nil {
		root = abs
	}
	return Config{
		Root:    root,
		Ignore:  append([]string(nil), defaultIgnore...),
		Openers: map[string][]string{},
	}
}

func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	Normalize(&cfg)
	return cfg, nil
}

func Save(path string, cfg Config) error {
	Normalize(&cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func Normalize(cfg *Config) {
	cfg.Root = Expand(cfg.Root)
	if abs, err := filepath.Abs(cfg.Root); err == nil {
		cfg.Root = abs
	}
	if cfg.Ignore == nil {
		cfg.Ignore = append([]string(nil), defaultIgnore...)
	}
	if cfg.Openers == nil {
		cfg.Openers = map[string][]string{}
	}
	for ext, command := range cfg.Openers {
		key := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), "."))
		delete(cfg.Openers, ext)
		if key == "" || len(command) == 0 {
			continue
		}
		cfg.Openers[key] = command
	}
}

func ConfigPath() string {
	if p := os.Getenv("HOMEWORK_CONFIG"); p != "" {
		return Expand(p)
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "homework-config.json"
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "homework", "config.json")
}

func DataPath() string {
	if p := os.Getenv("HOMEWORK_DATA"); p != "" {
		return Expand(p)
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "homework-data.json"
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "homework", "data.json")
}

// DefaultRoot picks the materials directory. The explicit environment
// variable wins; then the common Spanish/English Documents locations; then a
// folder next to the executable; finally the current directory.
func DefaultRoot() string {
	if root := os.Getenv("HOMEWORK_ROOT"); root != "" {
		return Expand(root)
	}

	home, _ := os.UserHomeDir()
	if home != "" {
		candidates := []string{
			filepath.Join(home, "Documentos", "homework"),
			filepath.Join(home, "Documents", "homework"),
			filepath.Join(home, "Documentos", "universidad"),
			filepath.Join(home, "Documents", "university"),
		}
		for _, candidate := range candidates {
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
		}
	}

	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		base := filepath.Base(dir)
		if base == "bin" {
			return filepath.Dir(dir)
		}
		if base == "go" || base == "python" {
			return filepath.Dir(dir)
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		return cwd
	}
	return "."
}

func Expand(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") || (runtime.GOOS == "windows" && strings.HasPrefix(path, `~\`)) {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func IsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

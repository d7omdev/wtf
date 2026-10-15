// Package config reads and writes the wtf configuration and locates state.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Config is the on-disk configuration (~/.config/wtf/config.json).
type Config struct {
	Provider   string            `json:"provider,omitempty"`
	Models     map[string]string `json:"models,omitempty"`
	TimeoutSec int               `json:"timeout_sec,omitempty"`
}

// ConfigPath returns the config file location, honoring overrides used in tests.
func ConfigPath() string {
	if p := os.Getenv("WTF_CONFIG"); p != "" {
		return p
	}
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "wtf", "config.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "wtf", "config.json")
}

// StateDir returns the base state directory (hooks write here, we read).
func StateDir() string {
	if d := os.Getenv("WTF_STATE_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return filepath.Join(d, "wtf")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "wtf")
}

// TTYLogPath is the pty recorder log written by `wtf shell`.
func TTYLogPath() string { return filepath.Join(StateDir(), "tty.log") }

// Load parses the config file; a missing file yields zero values.
func Load() Config {
	var c Config
	if b, err := os.ReadFile(ConfigPath()); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	if c.Models == nil {
		c.Models = map[string]string{}
	}
	return c
}

// Save writes the config atomically enough for a CLI (plain write).
func (c Config) Save() error {
	p := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(b, '\n'), 0o600)
}

// Timeout is the per-request provider timeout.
func (c Config) Timeout() time.Duration {
	if c.TimeoutSec > 0 {
		return time.Duration(c.TimeoutSec) * time.Second
	}
	return 90 * time.Second
}

// ModelFor returns the configured model for a provider ("" = provider default).
func (c Config) ModelFor(provider string) string { return c.Models[provider] }

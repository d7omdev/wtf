package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadRoundtrip(t *testing.T) {
	t.Setenv("WTF_CONFIG", filepath.Join(t.TempDir(), "config.json"))

	c := Load()
	if c.Provider != "" {
		t.Fatalf("expected empty config, got %+v", c)
	}
	if c.Timeout() != 90*time.Second {
		t.Errorf("default timeout = %v", c.Timeout())
	}

	c.Provider = "ollama"
	c.Models = map[string]string{"ollama": "qwen2.5-coder:7b"}
	c.TimeoutSec = 30
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}

	got := Load()
	if got.Provider != "ollama" || got.ModelFor("ollama") != "qwen2.5-coder:7b" || got.TimeoutSec != 30 {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
	if got.Timeout() != 30*time.Second {
		t.Errorf("timeout = %v", got.Timeout())
	}
}

func TestStateDirOverride(t *testing.T) {
	t.Setenv("WTF_STATE_DIR", "/tmp/whatever/wtf")
	if StateDir() != "/tmp/whatever/wtf" {
		t.Errorf("StateDir = %q", StateDir())
	}
	if TTYLogPath() != "/tmp/whatever/wtf/tty.log" {
		t.Errorf("TTYLogPath = %q", TTYLogPath())
	}
}

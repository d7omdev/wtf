// Package provider talks to the LLM backends: claude, codex, opencode (ACP),
// and ollama.
package provider

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Request is a single completion request.
type Request struct {
	System string
	Prompt string
	Model  string // "" = provider/config default
	Cwd    string // working directory to run the backend in
}

// Provider is one LLM backend.
type Provider interface {
	Name() string
	Complete(ctx context.Context, req Request) (string, error)
	// Models lists selectable model names (for the picker); discovery may fail
	// without failing the provider.
	Models(ctx context.Context) ([]string, error)
}

// ErrUnknown is returned by Get for an unsupported provider name.
var ErrUnknown = errors.New("unknown provider")

// Get resolves a provider by name.
func Get(name string) (Provider, error) {
	switch strings.ToLower(name) {
	case "claude":
		return claudeProvider{}, nil
	case "codex":
		return codexProvider{}, nil
	case "opencode":
		return opencodeProvider{}, nil
	case "ollama":
		return ollamaProvider{}, nil
	case "mock":
		return mockProvider{}, nil
	}
	return nil, fmt.Errorf("%w: %s (claude|codex|opencode|ollama)", ErrUnknown, name)
}

// Info describes an installed provider for the doctor and the picker.
type Info struct {
	Name string
	Path string // binary path, "" when not on PATH
	Note string // extra status, e.g. "3 models"
}

// All lists every provider with availability info.
func All() []Info {
	infos := []Info{
		{Name: "claude", Path: look("claude")},
		{Name: "codex", Path: look("codex")},
		{Name: "opencode", Path: look("opencode")},
		{Name: "ollama", Path: look("ollama")},
	}
	for i := range infos {
		if infos[i].Name == "ollama" && infos[i].Path != "" {
			if n, err := (ollamaProvider{}).count(context.Background()); err == nil {
				infos[i].Note = fmt.Sprintf("%d models", n)
			} else {
				infos[i].Note = "not running"
			}
		}
	}
	return infos
}

// Auto picks the first usable provider: claude, codex, opencode, ollama.
func Auto() string {
	for _, name := range []string{"claude", "codex", "opencode"} {
		if look(name) != "" {
			return name
		}
	}
	if look("ollama") != "" {
		return "ollama"
	}
	return ""
}

func look(bin string) string {
	p, err := exec.LookPath(bin)
	if err != nil {
		return ""
	}
	return p
}

func withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		d = 90 * time.Second
	}
	return context.WithTimeout(ctx, d)
}

// tail renders the last bytes of a backend's stderr for error messages.
func tail(buf interface{ String() string }) string {
	s := strings.TrimSpace(buf.String())
	if s == "" {
		return ""
	}
	if len(s) > 300 {
		s = "..." + s[len(s)-300:]
	}
	return ": " + s
}

package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type codexProvider struct{}

func (codexProvider) Name() string { return "codex" }

func (codexProvider) Complete(ctx context.Context, req Request) (string, error) {
	bin := look("codex")
	if bin == "" {
		return "", fmt.Errorf("codex CLI not found on PATH")
	}
	outFile, err := os.CreateTemp("", "wtf-codex-*")
	if err != nil {
		return "", err
	}
	outFile.Close()
	defer os.Remove(outFile.Name())

	args := []string{
		"exec",
		"--color", "never",
		"--skip-git-repo-check",
		"-s", "read-only",
		"-o", outFile.Name(),
	}
	if req.Model != "" {
		args = append(args, "-m", req.Model)
	}
	args = append(args, "-") // instructions from stdin

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = req.Cwd
	cmd.Stdin = strings.NewReader(req.System + "\n\n" + req.Prompt)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard // transcript noise; the answer comes via -o
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("codex: %s%s", err, tail(&stderr))
	}

	b, err := os.ReadFile(outFile.Name())
	if err != nil {
		return "", fmt.Errorf("codex: read output: %w", err)
	}
	out := strings.TrimSpace(string(b))
	if out == "" {
		return "", fmt.Errorf("codex returned an empty response%s", tail(&stderr))
	}
	return out, nil
}

func (codexProvider) Models(ctx context.Context) ([]string, error) {
	// codex resolves "default" from the user's ~/.codex/config.toml
	return []string{"default"}, nil
}

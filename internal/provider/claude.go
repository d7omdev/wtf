package provider

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type claudeProvider struct{}

func (claudeProvider) Name() string { return "claude" }

func (claudeProvider) Complete(ctx context.Context, req Request) (string, error) {
	bin := look("claude")
	if bin == "" {
		return "", fmt.Errorf("claude CLI not found on PATH")
	}

	out, errText, err := claudeRun(ctx, bin, req, true)
	if err != nil && isUnknownFlag(errText) {
		// older claude without --restricted
		out, errText, err = claudeRun(ctx, bin, req, false)
	}
	if err != nil {
		return "", fmt.Errorf("claude: %s%s", err, tail(stringer(errText)))
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return "", fmt.Errorf("claude returned an empty response%s", tail(stringer(errText)))
	}
	return out, nil
}

func claudeRun(ctx context.Context, bin string, req Request, restricted bool) (out, errText string, err error) {
	args := []string{"-p", "--output-format", "text"}
	if restricted {
		args = append(args, "--restricted")
	}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	if req.System != "" {
		args = append(args, "--append-system-prompt", req.System)
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = req.Cwd
	cmd.Stdin = strings.NewReader(req.Prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stdout.String(), stderr.String(), err
}

func isUnknownFlag(errText string) bool {
	l := strings.ToLower(errText)
	return strings.Contains(l, "unknown option") ||
		strings.Contains(l, "unknown flag") ||
		strings.Contains(l, "unexpected argument")
}

func (claudeProvider) Models(ctx context.Context) ([]string, error) {
	// aliases accepted by --model; full model IDs also work via "custom…"
	return []string{"sonnet", "opus", "haiku", "fable"}, nil
}

// stringer adapts a plain string to the tail() signature.
type stringer string

func (s stringer) String() string { return string(s) }

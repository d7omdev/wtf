// Package facts assembles the context block handed to the model.
package facts

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"wtf/internal/state"
	"wtf/internal/trace"
)

// Facts is everything wtf knows about the last command.
type Facts struct {
	Cmd      string
	Exit     int
	OK       bool
	Duration time.Duration
	Cwd      string
	Git      string // "main, clean" / "main, 3 modified" / "not a git repo"
	Trace    string // cleaned, tail-trimmed output; "" when not captured
}

// Gather builds facts from recorded state (git info is probed now, on demand,
// so the per-prompt hooks stay cheap).
func Gather(l *state.Last) Facts {
	return Facts{
		Cmd:      l.Cmd,
		Exit:     l.Exit,
		OK:       l.Exit == 0,
		Duration: l.Duration(),
		Cwd:      l.Cwd,
		Git:      gitInfo(l.Cwd),
		Trace:    trace.ForModel(l.Trace(), l.Cmd),
	}
}

// DurationStr formats the duration, or "" when unknown.
func (f Facts) DurationStr() string {
	if f.Duration <= 0 {
		return ""
	}
	return fmt.Sprintf("%.2fs", f.Duration.Seconds())
}

func gitInfo(cwd string) string {
	if cwd == "" || runGit(cwd, 400*time.Millisecond, "rev-parse", "--is-inside-work-tree") == "" {
		return "not a git repo"
	}
	branch := runGit(cwd, 400*time.Millisecond, "rev-parse", "--abbrev-ref", "HEAD")
	switch branch {
	case "":
		branch = "unknown branch"
	case "HEAD":
		branch = "detached HEAD"
	}
	status := runGit(cwd, 600*time.Millisecond, "status", "--porcelain", "-uno")
	if strings.TrimSpace(status) == "" {
		return branch + ", clean"
	}
	n := len(strings.Split(strings.TrimSpace(status), "\n"))
	return fmt.Sprintf("%s, %d modified", branch, n)
}

func runGit(cwd string, d time.Duration, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", cwd}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

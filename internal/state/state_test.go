package state

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeState simulates what the shell hooks write for one terminal.
func writeState(t *testing.T, root, tty, cmd, exit, trace string) string {
	t.Helper()
	d := filepath.Join(root, "state", tty)
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"cmd":   cmd,
		"exit":  exit,
		"cwd":   "/home/user/proj",
		"start": "1000",
		"end":   "2500",
		"rec":   "0 0",
		"trace": trace,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(d, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

func TestReadFallsBackToNewestState(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WTF_STATE_DIR", root)
	writeState(t, root, "_dev_pts_9", "older command", "1", "old")
	writeState(t, root, "_dev_pts_7", "npm test", "1", "some output")

	l, err := Read()
	if err != nil {
		// Test stdin may match a dir — but we wrote only these two; newest wins.
		t.Fatal(err)
	}
	if l.Cmd == "" || l.Exit != 1 {
		t.Errorf("Read = %+v", l)
	}
	if l.Cwd != "/home/user/proj" {
		t.Errorf("cwd = %q", l.Cwd)
	}
	if d := l.Duration(); d.Milliseconds() != 1500 {
		t.Errorf("duration = %v", d)
	}
}

func TestReadMissingIsErrNoState(t *testing.T) {
	t.Setenv("WTF_STATE_DIR", t.TempDir())
	_, err := Read()
	if !errors.Is(err, ErrNoState) {
		t.Errorf("err = %v, want ErrNoState", err)
	}
}

func TestEmptyCmdIsErrNoState(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WTF_STATE_DIR", root)
	d := filepath.Join(root, "state", "_dev_pts_1")
	os.MkdirAll(d, 0o755)
	os.WriteFile(filepath.Join(d, "cmd"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(d, "exit"), []byte("0"), 0o644)

	if _, err := Read(); !errors.Is(err, ErrNoState) {
		t.Errorf("err = %v, want ErrNoState", err)
	}
}

func TestTracePrefersRecorderSlice(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WTF_STATE_DIR", root)

	// pty recorder log with two "command outputs"
	log := filepath.Join(root, "tty.log")
	os.WriteFile(log, []byte("FIRST_OUTPUT\nSECOND_OUTPUT\n"), 0o644)
	// slice covering only the second line: len("FIRST_OUTPUT\n") = 13
	d := writeState(t, root, "_dev_pts_2", "echo hi", "0", "tmux trace fallback")
	os.WriteFile(filepath.Join(d, "rec"), []byte("13 27"), 0o644)

	l, err := Read()
	if err != nil {
		t.Fatal(err)
	}
	if got := l.Trace(); got != "SECOND_OUTPUT\n" {
		t.Errorf("Trace = %q, want recorder slice", got)
	}

	// invalid slice (stale offsets) → falls back to the tmux trace file
	os.WriteFile(filepath.Join(d, "rec"), []byte("0 999999"), 0o644)
	l, _ = Read()
	if got := l.Trace(); got != "tmux trace fallback" {
		t.Errorf("Trace = %q, want tmux fallback", got)
	}
}

func TestSanitize(t *testing.T) {
	if got := sanitize("/dev/pts/3"); got != "_dev_pts_3" {
		t.Errorf("sanitize = %q", got)
	}
}

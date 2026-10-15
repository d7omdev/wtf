// Package state reads the per-command state recorded by the shell hooks.
//
// The hooks (shell/wtf.zsh, shell/wtf.bash) write one directory per terminal:
//
//	$XDG_STATE_HOME/wtf/state/<tty>/cmd     the command line
//	$XDG_STATE_HOME/wtf/state/<tty>/exit    exit code
//	$XDG_STATE_HOME/wtf/state/<tty>/cwd     working directory
//	$XDG_STATE_HOME/wtf/state/<tty>/start   epoch millis when it started
//	$XDG_STATE_HOME/wtf/state/<tty>/end     epoch millis when it finished
//	$XDG_STATE_HOME/wtf/state/<tty>/rec     pty-recorder offsets "off0 off1"
//	$XDG_STATE_HOME/wtf/state/<tty>/trace   tmux pane capture (optional)
package state

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"wtf/internal/config"
)

// ErrNoState means no command has been recorded yet (hooks not installed).
var ErrNoState = errors.New("no command recorded yet")

// Last describes the most recent shell command as captured by the hooks.
type Last struct {
	Cmd     string
	Exit    int
	Cwd     string
	StartMS int64 // epoch millis; 0 = unknown
	EndMS   int64
	Rec0    int64 // byte offsets into the pty recorder log; Rec0 < Rec1 = valid
	Rec1    int64
	dir     string
}

// Duration returns how long the command ran, or 0 when unknown.
func (l *Last) Duration() time.Duration {
	if l.StartMS > 0 && l.EndMS > l.StartMS {
		return time.Duration(l.EndMS-l.StartMS) * time.Millisecond
	}
	return 0
}

// TTYID returns a sanitized identifier for the controlling terminal.
func TTYID() string {
	target, err := os.Readlink("/proc/self/fd/0")
	if err != nil || !strings.HasPrefix(target, "/dev/") {
		return ""
	}
	return sanitize(target)
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, s)
}

func stateRoot() string { return filepath.Join(config.StateDir(), "state") }

// Read loads the state recorded for the calling terminal, falling back to the
// most recently written terminal if this one has none.
func Read() (*Last, error) {
	var dirs []string
	if id := TTYID(); id != "" {
		dirs = append(dirs, filepath.Join(stateRoot(), id))
	}

	type cand struct {
		dir  string
		time time.Time
	}
	var cands []cand
	if entries, err := os.ReadDir(stateRoot()); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			d := filepath.Join(stateRoot(), e.Name())
			if st, err := os.Stat(filepath.Join(d, "cmd")); err == nil {
				cands = append(cands, cand{d, st.ModTime()})
			}
		}
		sort.Slice(cands, func(i, j int) bool { return cands[i].time.After(cands[j].time) })
	}
	for _, c := range cands {
		dirs = append(dirs, c.dir)
	}

	for _, d := range dirs {
		if l, err := readDir(d); err == nil {
			return l, nil
		}
	}
	return nil, ErrNoState
}

func readDir(d string) (*Last, error) {
	b, err := os.ReadFile(filepath.Join(d, "cmd"))
	if err != nil || len(strings.TrimSpace(string(b))) == 0 {
		return nil, ErrNoState
	}
	l := &Last{Cmd: strings.TrimRight(string(b), "\n"), dir: d}
	if v, err := readNum(filepath.Join(d, "exit")); err == nil {
		l.Exit = int(v)
	} else {
		return nil, ErrNoState
	}
	if s, err := os.ReadFile(filepath.Join(d, "cwd")); err == nil {
		l.Cwd = string(s)
	}
	if v, err := readNum(filepath.Join(d, "start")); err == nil {
		l.StartMS = v
	}
	if v, err := readNum(filepath.Join(d, "end")); err == nil {
		l.EndMS = v
	}
	if s, err := os.ReadFile(filepath.Join(d, "rec")); err == nil {
		if parts := strings.Fields(string(s)); len(parts) == 2 {
			l.Rec0, _ = strconv.ParseInt(parts[0], 10, 64)
			l.Rec1, _ = strconv.ParseInt(parts[1], 10, 64)
		}
	}
	return l, nil
}

func readNum(p string) (int64, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return 0, strconv.ErrSyntax
	}
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v, nil
	}
	// defensive: older hooks wrote float seconds/millis
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int64(f), nil
}

// Trace returns the raw captured output of the command, if any. It prefers the
// exact pty-recorder slice and falls back to the tmux pane capture.
func (l *Last) Trace() string {
	if l.Rec1 > l.Rec0 {
		if s := readSlice(config.TTYLogPath(), l.Rec0, l.Rec1); s != "" {
			return s
		}
	}
	if b, err := os.ReadFile(filepath.Join(l.dir, "trace")); err == nil {
		return string(b)
	}
	return ""
}

const maxTraceSlice = 1 << 20 // 1 MiB

func readSlice(path string, off0, off1 int64) string {
	if off1-off0 > maxTraceSlice {
		off0 = off1 - maxTraceSlice
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || off0 < 0 || off1 > st.Size() {
		return ""
	}
	buf := make([]byte, off1-off0)
	if _, err := f.ReadAt(buf, off0); err != nil {
		return ""
	}
	return string(buf)
}

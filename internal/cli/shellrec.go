package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	"golang.org/x/term"

	"wtf/internal/config"
	"wtf/internal/state"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Run your shell with full output recording (exact traces for wtf)",
	Long: `Starts your $SHELL inside a pty that tees all input and output to a log.
While you work this way, wtf gets the EXACT output of your last command —
no tmux needed. Exit the shell to return to your normal session.`,
	SilenceUsage: true,
	RunE:         runRecorder,
}

const maxLogSize = 8 << 20 // rotate the recorder log above this

func runRecorder(cmd *cobra.Command, args []string) error {
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/sh"
	}
	logPath := config.TTYLogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}
	if st, err := os.Stat(logPath); err == nil && st.Size() > maxLogSize {
		_ = os.Rename(logPath, logPath+".old")
	}
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer logf.Close()

	c := exec.Command(sh)
	c.Env = append(os.Environ(), "WTF_RECORDER_LOG="+logPath)

	master, err := pty.StartWithSize(c, winsize())
	if err != nil {
		return fmt.Errorf("start %s: %w", sh, err)
	}
	defer master.Close()

	// raw terminal so the child gets every keystroke (incl. Ctrl-C as data)
	var oldState *term.State
	if s, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
		oldState = s
	}
	restore := func() {
		if oldState != nil {
			_ = term.Restore(int(os.Stdin.Fd()), oldState)
			oldState = nil
		}
	}
	defer restore()

	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	go func() {
		for range winch {
			_ = pty.Setsize(master, winsize())
		}
	}()

	// child output → this terminal and the log
	go func() { _, _ = io.Copy(io.MultiWriter(os.Stdout, logf), master) }()

	// this terminal's input → child
	_, _ = io.Copy(master, os.Stdin)

	err = c.Wait()
	restore()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		os.Exit(ee.ExitCode())
	}
	return nil
}

func winsize() *pty.Winsize {
	if ws, err := unix.IoctlGetWinsize(int(os.Stdin.Fd()), unix.TIOCGWINSZ); err == nil && ws.Row > 0 {
		return &pty.Winsize{Rows: ws.Row, Cols: ws.Col}
	}
	return &pty.Winsize{Rows: 24, Cols: 80}
}

// newestStateTime reports when any terminal's state was last written.
func newestStateTime() (time.Time, error) {
	root := filepath.Join(config.StateDir(), "state")
	entries, err := os.ReadDir(root)
	if err != nil {
		return time.Time{}, err
	}
	var newest time.Time
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if st, err := os.Stat(filepath.Join(root, e.Name(), "cmd")); err == nil && st.ModTime().After(newest) {
			newest = st.ModTime()
		}
	}
	if newest.IsZero() {
		return time.Time{}, state.ErrNoState
	}
	return newest, nil
}

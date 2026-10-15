// Package color renders styled terminal output, honoring NO_COLOR and
// non-TTY output automatically via lipgloss.
package color

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var r = lipgloss.NewRenderer(os.Stdout)

var (
	// OK marks success (green, bold).
	OK = r.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	// FAIL marks failure (red, bold).
	FAIL = r.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	// WARN marks warnings (yellow, bold).
	WARN = r.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	// Dim is secondary metadata.
	Dim = r.NewStyle().Foreground(lipgloss.Color("8"))
	// Cmd renders shell commands (green, bold).
	Cmd = r.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	// Accent highlights small details like the git branch (blue).
	Accent = r.NewStyle().Foreground(lipgloss.Color("12"))
	// Msg is the default body style.
	Msg = r.NewStyle()
)

// Disable strips all styling (--no-color).
func Disable() { r.SetColorProfile(termenv.Ascii) }

// Error prints an error line to stderr in the failure style.
func Error(msg string) { fmt.Fprintln(os.Stderr, FAIL.Render(msg)) }

// Warn prints a warning line to stderr.
func Warn(msg string) { fmt.Fprintln(os.Stderr, WARN.Render(msg)) }

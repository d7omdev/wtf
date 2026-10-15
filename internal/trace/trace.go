// Package trace cleans captured terminal output for the model's eyes.
package trace

import (
	"regexp"
	"strings"
)

var ansiRE = regexp.MustCompile(
	"\x1b\\[[0-9;:?]*[ -/]*[@-~]" + // CSI sequences
		"|\x1b\\][^\x07\x1b]*(?:\x07|\x1b\\\\)" + // OSC sequences
		"|\x1b[P^_][^\x1b]*(?:\x1b\\\\)" + // DCS/SOS/PM
		"|\x1b[@-Z\\\\-_]", // other two-byte escapes
)

// Clean removes ANSI escapes, resolves carriage-return overwrites (progress
// bars), and drops stray control characters.
func Clean(s string) string {
	s = ansiRE.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if j := strings.LastIndex(line, "\r"); j >= 0 {
			line = line[j+1:]
		}
		lines[i] = stripControls(line)
	}
	return strings.Join(lines, "\n")
}

func stripControls(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || (r >= 0x20 && r != 0x7f) {
			return r
		}
		return -1
	}, s)
}

// promptLineRE matches bare shell prompt lines like "user@host:~$" or "❯ ".
var promptLineRE = regexp.MustCompile(`^[^\s]*[\$%#>❯]$`)

// CutAboveCommand drops everything up to and including the last line that
// echoes the command (a tmux capture contains the prompt and typed line too),
// plus any fresh prompt drawn after the command.
func CutAboveCommand(s, cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return s
	}
	probe := cmd
	if len(probe) > 60 {
		probe = probe[:60]
	}
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], probe) {
			lines = lines[i+1:]
			break
		}
	}
	// drop trailing prompt lines (the shell may have drawn a new one)
	for len(lines) > 0 && promptLineRE.MatchString(strings.TrimSpace(lines[len(lines)-1])) {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// TrimTail keeps the last maxLines lines / maxBytes of output — the end is
// what matters for failures.
func TrimTail(s string, maxLines, maxBytes int) string {
	s = strings.TrimRight(s, "\n")
	lines := strings.Split(s, "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
		s = "(earlier output omitted)\n" + strings.Join(lines, "\n")
	} else {
		s = strings.Join(lines, "\n")
	}
	if len(s) > maxBytes {
		s = "(earlier output omitted)\n" + s[len(s)-maxBytes:]
	}
	return s
}

// ForModel runs the full pipeline from raw capture to prompt-ready trace.
func ForModel(raw, cmd string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	s := CutAboveCommand(Clean(raw), cmd)
	return strings.TrimSpace(TrimTail(s, 70, 6000))
}

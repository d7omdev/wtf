// Package prompt builds the model prompt and post-processes its answer into
// terse terminal text. Brevity is enforced structurally: hard limits in the
// prompt plus clamping in post-processing.
package prompt

import (
	"fmt"
	"strings"

	"wtf/internal/facts"
)

// Mode selects the request type.
type Mode int

const (
	// StatusOK reports a successful command.
	StatusOK Mode = iota
	// StatusFail explains a failed command in one or two sentences.
	StatusFail
	// Explain gives 2-3 sentences: what happened, why, what to do.
	Explain
	// Fix demands a single corrected command line.
	Fix
)

const systemBase = `You are "wtf", a terse terminal diagnostician. You receive the user's LAST SHELL COMMAND, its exit status, and captured terminal output (trace). Rules: plain text only — no markdown, no code fences, no bullet lists, no emoji, no ANSI colors. Never invent output you cannot see. Never repeat the command back. Be concrete and correct; if unsure, give the most likely cause.`

var modeInstruction = map[Mode]string{
	StatusOK:   `Reply in EXACTLY ONE sentence (at most 20 words): what the command accomplished, based on the trace. Start with the result, not with "The command".`,
	StatusFail: `Reply in ONE OR TWO sentences (at most 25 words total): the concrete reason the command failed, naming the specific error. If it actually succeeded despite the exit code, say so in one sentence.`,
	Explain:    `Reply in 2-3 SHORT sentences (at most 60 words): what happened, the root cause in plain language, and what to do about it.`,
	Fix:        `Output ONLY the corrected shell command: one line, runnable as-is. No markdown, no quotes, no explanation, no prompt symbol, no trailing punctuation. Change only what is necessary to make it work. If no fix is possible, output exactly: WTF_UNFIXABLE`,
}

// Build returns the system prompt and the user message for a request.
func Build(m Mode, f facts.Facts) (system, user string) {
	system = systemBase + "\n\n" + modeInstruction[m]

	var b strings.Builder
	fmt.Fprintf(&b, "command: %s\n", f.Cmd)
	fmt.Fprintf(&b, "exit code: %d (%s)\n", f.Exit, okFail(f.OK))
	if d := f.DurationStr(); d != "" {
		fmt.Fprintf(&b, "duration: %s\n", d)
	}
	if f.Cwd != "" {
		fmt.Fprintf(&b, "cwd: %s\n", f.Cwd)
	}
	if f.Git != "" {
		fmt.Fprintf(&b, "git: %s\n", f.Git)
	}
	if f.Trace != "" {
		b.WriteString("trace (oldest first, most recent last):\n")
		b.WriteString(f.Trace)
	} else {
		b.WriteString("trace: not captured — reason from the command and exit code alone\n")
	}
	return system, b.String()
}

func okFail(ok bool) string {
	if ok {
		return "ok"
	}
	return "failed"
}

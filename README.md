# wtf

A small, fast CLI: why did that last command fail (or succeed), and how do I fix it?

`wtf` reads state captured by shell hooks at command time (command, exit code, duration,
cwd, git branch, and the real output trace) and asks an LLM to explain it in one or
two sentences. `wtf fix` prints a corrected command, then runs it after a y/N confirmation.

## Install

```sh
go build -o ~/.local/bin/wtf ./cmd/wtf
# zsh:
eval "$(wtf init zsh)"        # add this to ~/.zshrc
# bash:
eval "$(wtf init bash)"       # add this to ~/.bashrc
```

Use tmux or `wtf shell` for exact output traces. Run `wtf doctor` to verify providers and hooks.

## Commands

- `wtf` - did the last command succeed or fail, and why (1-2 sentences)
- `wtf explain` - same, in 2-3 sentences
- `wtf fix` - print the corrected command, run it after y/N
- `wtf use [provider] [model]` - interactive picker (or set directly with args)
- `wtf init zsh|bash` - print the shell hooks for eval
- `wtf shell` - run your shell with full output recording
- `wtf doctor` - health check for providers and hooks
- `wtf version`

## Providers

Claude Code (`claude -p --restricted`), Codex (`codex exec`), OpenCode (`opencode acp`,
hand-rolled JSON-RPC client), Ollama (`POST /api/chat`), plus in-memory `mock` for testing.

## Enforcing brevity

Limits: status = 2 sentences / 25 words, explain = 3 sentences / 60 words, fix = one line.
Status/explain answers are clamped; markdown/quotes stripped.

## Color

Colors respect `NO_COLOR` and non-TTY (piped) output via lipgloss.

## No Python

This project is Go only.

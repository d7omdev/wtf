# AGENTS.md

## What this is

`wtf` — a CLI that inspects your **last shell command**: did it succeed/fail, why, and how to fix it. The repo is brand new (no code yet). The decisions below were made with the user — **do not re-ask them**; implement them.

## Product spec (agreed — do not relitigate)

- `wtf` → status of the last command in **one or two concise, terminal-appropriate sentences** (success: say it succeeded and what happened; error: say why).
- `wtf fix` → produce the corrected command, **print it, then run it after a y/N confirmation** (user explicitly chose this over inserting at the prompt buffer).
- `wtf explain` → what happened and why; still terse, 2–3 sentences max.
- **Color output is required** — respect `NO_COLOR` and non-TTY (pipe) output.
- **Interactive picker** to choose provider + model and persist them as the default (e.g. `wtf use`).
- Enforce brevity structurally (hard prompt limits + post-processing), don't trust the model to stay short.

## Stack (decided)

- **Go**, single static binary. **User explicitly rejected Python** — do not propose it.
- Provider backends:
  - `claude -p` (headless print mode)
  - `codex exec`
  - `opencode acp` — ACP is JSON-RPC 2.0 over stdio using **newline-delimited JSON** (no Content-Length framing). Flow: `initialize` → `session/new` → `session/prompt`, handle `session/update` notifications. No official Go SDK (TS/Rust only) — hand-roll the ~150-line client.
  - `ollama` — `POST /api/chat` with `"stream": false` (default `localhost:11434`); list models via `GET /api/tags`.

## Environment (verified on this machine)

- zsh 5.9 is the user's shell (bash also present); `tmux`, `jq`, `git` on PATH.
- `claude`, `codex`, `opencode` CLIs installed (`~/.local/bin/claude`, `/usr/bin/codex`, `~/.opencode/bin/opencode`).
- Node via `~/.vite-plus/bin/node`; Python 3.14 exists but must not be used for this project.
- Not a git repository yet.

## Gotchas

- **Output capture must happen at hook time.** After a command finishes, the shell cannot recover its stdout/stderr. The shell integration (zsh `preexec`/`precmd`, bash equivalent) must record command + exit code + trace when the command runs — via tmux `capture-pane` history slicing or an offset-based PTY recorder — never try to reconstruct output later.
- Keep the `wtf` invocation fast: state is written by shell hooks (plain files, no JSON escaping games in shell); the Go binary reads it on demand.
- When build/test/lint commands exist, record the exact invocations here (this section is a placeholder until then).

## Build / test / lint

- Build: `go build -o bin/wtf ./cmd/wtf` (or `make build`)
- Test: `go test ./...` (`make test` runs `-race`)
- Vet: `go vet ./...` (`make vet`)
- Format: `gofmt -w .` (`make fmt`) — keep clean; CI/Lint check uses `gofmt -l .`
- Makefile: `make build|test|vet|fmt|lint|install|doctor`

## Embedding
- `wtf init zsh|bash` reads embedded scripts in `shell/` (package `shell`) built via `//go:embed`.

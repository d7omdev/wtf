# wtf — build status

- All core source builds clean; `go vet && go test ./...` green (config, prompt, provider, state, trace).
- Commands: root (status), explain, fix (print + y/N confirm), use (huh picker + direct args), init (zsh/bash hooks), doctor, shell (pty recorder), version. Mock provider for tests.
- Providers: claude (real API untested live), codex (untested live), opencode ACP (end-to-end with fake agent + permission flow), ollama (live: works).
- Hooks verified in tmux with real zsh: per-tty state, exit code, cwd, duration (integer ms), trace via capture-pane (CutAboveCommand fixes powerline noise), rec offsets for `wtf shell`.
- Pending: live tests of claude/codex/opencode against real CLIs; git push; install binary to ~/.local/bin/wtf; shellcheck on hooks.

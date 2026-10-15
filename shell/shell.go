// Package shell embeds the shell integration scripts emitted by `wtf init`.
package shell

import _ "embed"

// ZSH is the zsh hook script (preexec/precmd state capture, tmux trace,
// pty-recorder offsets).
//
//go:embed wtf.zsh
var ZSH string

// BASH is the bash hook script (PROMPT_COMMAND state capture, tmux trace).
//
//go:embed wtf.bash
var BASH string

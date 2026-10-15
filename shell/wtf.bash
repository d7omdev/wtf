# wtf shell integration (bash) — records the last command, exit code, and cwd.
# Traces need tmux (capture-pane); exact recorder traces need zsh or `wtf shell`.
# Install with:  eval "$(wtf init bash)"
[ -n "${BASH_VERSION:-}" ] || return 0

__wtf_base="${WTF_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/wtf}"
__wtf_tty="$(command tty 2>/dev/null)"
__wtf_tty="${__wtf_tty//[^0-9A-Za-z]/_}"
[ -n "$__wtf_tty" ] || __wtf_tty=default
__wtf_dir="$__wtf_base/state/$__wtf_tty"

__wtf_prompt() {
  local ec=$?
  local cmd
  cmd="$(command fc -ln -1 2>/dev/null)"
  cmd="${cmd#"${cmd%%[![:space:]]*}"}" # ltrim
  [ -n "$cmd" ] || return 0
  command mkdir -p -- "$__wtf_dir" 2>/dev/null || return 0
  printf '%s' "$cmd" > "$__wtf_dir/cmd"
  printf '%s' "$ec"  > "$__wtf_dir/exit"
  printf '%s' "$PWD" > "$__wtf_dir/cwd"
  : > "$__wtf_dir/start"
  : > "$__wtf_dir/end"
  printf '0 0' > "$__wtf_dir/rec"
  if [ -n "${TMUX:-}" ] && [ -n "${TMUX_PANE:-}" ]; then
    command tmux capture-pane -p -t "$TMUX_PANE" -S -400 2>/dev/null > "$__wtf_dir/trace" || : > "$__wtf_dir/trace"
  else
    : > "$__wtf_dir/trace"
  fi
}

# install as the first PROMPT_COMMAND entry so $? is the real command status
case ";${PROMPT_COMMAND[*]-};" in
  *";__wtf_prompt;"*) ;;
  *)
    if declare -p PROMPT_COMMAND 2>/dev/null | command grep -q 'declare -a'; then
      PROMPT_COMMAND=(__wtf_prompt "${PROMPT_COMMAND[@]}")
    else
      PROMPT_COMMAND="__wtf_prompt${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
    fi
    ;;
esac

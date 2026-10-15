# wtf shell integration (zsh) — records the last command, exit code, and output
# so `wtf` can explain it. Install with:  eval "$(wtf init zsh)"
() {
  [[ -o interactive ]] || return 0

  local base="${WTF_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/wtf}"
  local tty_id="${TTY:-}"
  [[ -z $tty_id ]] && tty_id="$(command tty 2>/dev/null)"
  tty_id="${tty_id//[^0-9A-Za-z]/_}"
  [[ -z $tty_id ]] && tty_id="default"

  typeset -g __wtf_base="$base"
  typeset -g __wtf_dir="$base/state/$tty_id"
  typeset -g __wtf_cmd="" __wtf_t0="" __wtf_off0=0 __wtf_skip=0

  zmodload zsh/datetime 2>/dev/null
  zmodload -F zsh/stat b:zstat 2>/dev/null

  __wtf_preexec() {
    # Never record wtf's own invocations (keeps prior command analysable).
    case "$1" in
      wtf|wtf\ *|*/wtf|*/wtf\ *)
        __wtf_skip=1
        return 0
        ;;
    esac
    __wtf_skip=0
    __wtf_cmd="$1"
    __wtf_t0=""
    if (( ${+EPOCHREALTIME} )); then
      __wtf_t0=$(( EPOCHREALTIME * 1000 ))
      __wtf_t0="${__wtf_t0%%.*}" # whole milliseconds
    fi
    __wtf_off0=0
    if [[ -f "$__wtf_base/tty.log" ]] && (( ${+builtins[zstat]} )); then
      local -a __wtf_sz
      if zstat -A __wtf_sz +size -- "$__wtf_base/tty.log" 2>/dev/null; then
        __wtf_off0=$(( __wtf_sz[1] ))
      fi
    fi
  }

  __wtf_precmd() {
    local ec=$?
    if (( ${__wtf_skip:-0} == 1 )); then
      __wtf_skip=0
      return 0
    fi
    local d="$__wtf_dir"
    if [[ ! -d $d ]]; then
      command mkdir -p -- "$d" 2>/dev/null || return 0
    fi

    local end=""
    if (( ${+EPOCHREALTIME} )); then
      end=$(( EPOCHREALTIME * 1000 ))
      end="${end%%.*}" # whole milliseconds
    fi
    print -rn -- "$__wtf_cmd"  > "$d/cmd"
    print -rn -- "$ec"         > "$d/exit"
    print -rn -- "$PWD"        > "$d/cwd"
    print -rn -- "$__wtf_t0"   > "$d/start"
    print -rn -- "$end"        > "$d/end"

    # exact output slice from the pty recorder (wtf shell), when active
    local rec0=0 rec1=0
    if (( __wtf_off0 > 0 )) && [[ -f "$__wtf_base/tty.log" ]] && (( ${+builtins[zstat]} )); then
      local -a __wtf_sz
      if zstat -A __wtf_sz +size -- "$__wtf_base/tty.log" 2>/dev/null; then
        (( __wtf_sz[1] > __wtf_off0 )) && { rec0=$__wtf_off0; rec1=$(( __wtf_sz[1] )); }
      fi
    fi
    print -rn -- "$rec0 $rec1" > "$d/rec"

    # tmux pane capture (works without the recorder; wtf keeps the tail)
    if [[ -n $TMUX && -n $TMUX_PANE ]]; then
      command tmux capture-pane -p -t "$TMUX_PANE" -S -400 2>/dev/null > "$d/trace" || : > "$d/trace"
    else
      : > "$d/trace"
    fi
  }

  preexec_functions+=(__wtf_preexec)
  precmd_functions+=(__wtf_precmd)
}

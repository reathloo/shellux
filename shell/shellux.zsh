# Shellux integration for Zsh.
# Source this file from .zshrc after installing the shellux binary.

if (( $+functions[shellux_prompt] == 0 || ${shellux_integration_version:-0} < 20 )); then
  typeset -g shellux_integration_version=20
  typeset -g shellux_enabled="${shellux_enabled:-1}"
  typeset -g shellux_render_header="${shellux_render_header:-1}"
  typeset -g shellux_clear_startup_scrollback="${shellux_clear_startup_scrollback:-1}"
  typeset -g shellux_preserve_content="${shellux_preserve_content:-0}"
  typeset -g shellux_output_mode="${shellux_output_mode:-0}"
  typeset -g shellux_command_paused="${shellux_command_paused:-0}"
  typeset -g shellux_watch_pid="${shellux_watch_pid:-}"
  typeset -g shellux_watch_before_command=0

  shellux_set_title() {
    # OSC 0 names both the terminal window and its tab. Never emit it to pipes.
    [[ -t 1 ]] || return 0
    printf '\033]0;Shellux\007'
  }

  shellux_start_watch() {
    [[ -t 1 ]] || return 0
    if [[ -n "$shellux_watch_pid" ]] && kill -0 "$shellux_watch_pid" 2>/dev/null; then
      return 0
    fi
    command "${SHELLUX_BIN:-shellux}" --watch --in-place --parent-pid $$ </dev/null >/dev/tty 2>/dev/null &!
    shellux_watch_pid=$!
  }

  shellux_prompt() {
    (( shellux_enabled )) || return 0
    shellux_set_title
    if (( shellux_output_mode )); then
      # Normal output keeps the full viewport until an explicit reload.
      shellux_render_header=0
      shellux_command_paused=0
      shellux_watch_before_command=0
      return 0
    fi
    whence -p "${SHELLUX_BIN:-shellux}" >/dev/null 2>&1 || return 0
    if (( shellux_render_header )); then
      if (( shellux_clear_startup_scrollback )); then
        local shellux_help
        shellux_help=$(command "${SHELLUX_BIN:-shellux}" --help 2>&1)
        if [[ "$shellux_help" == *"clear-scrollback"* ]]; then
          command "${SHELLUX_BIN:-shellux}" --once --reserve-header --clear-scrollback || return $?
        else
          command "${SHELLUX_BIN:-shellux}" --once --reserve-header || return $?
        fi
      elif (( shellux_preserve_content )); then
        command "${SHELLUX_BIN:-shellux}" --once --reserve-header --preserve-content || return $?
      else
        command "${SHELLUX_BIN:-shellux}" --once --reserve-header || return $?
      fi
      shellux_render_header=0
      shellux_clear_startup_scrollback=0
    fi
    shellux_preserve_content=0
    if (( ! shellux_output_mode )); then
      shellux_start_watch
    fi
    shellux_command_paused=0
    shellux_watch_before_command=0
  }
  shellux_stop_watch() {
    if [[ -n "$shellux_watch_pid" ]] && kill -0 "$shellux_watch_pid" 2>/dev/null; then
      kill -TERM "$shellux_watch_pid" 2>/dev/null || true
      for _ in {1..100}; do
        if ! kill -0 "$shellux_watch_pid" 2>/dev/null; then
          break
        fi
        sleep 0.01
      done
      if kill -0 "$shellux_watch_pid" 2>/dev/null; then
        print -u2 'shellux: live renderer did not stop in time'
        return 1
      fi
    fi
    shellux_watch_pid=""
  }
  shellux_reload() {
    if (( ! shellux_enabled )); then
      print -u2 "shellux: header is off; run 'shellux on'"
      return 1
    fi
    shellux_set_title
    shellux_stop_watch || return $?
    # The renderer clears history only after it has drawn the new header.
    # Clearing it first makes some terminals retain one blank history line.
    shellux_render_header=1
    command "${SHELLUX_BIN:-shellux}" --once --reserve-header --clear-scrollback || return $?
    shellux_render_header=0
    shellux_clear_startup_scrollback=0
    shellux_preserve_content=0
    shellux_output_mode=0
    shellux_start_watch
  }
  shellux_off() {
    (( shellux_enabled )) || return 0
    shellux_stop_watch || return $?
    shellux_enabled=0
    shellux_render_header=0
    shellux_clear_startup_scrollback=0
    shellux_preserve_content=0
    shellux_output_mode=0
    shellux_command_paused=0
    shellux_watch_before_command=0
    printf '\033[r\033[H\033[2J'
  }
  shellux_on() {
    (( shellux_enabled )) && return 0
    shellux_enabled=1
    shellux_reload && return 0
    local result=$?
    shellux_enabled=0
    return "$result"
  }
  shellux_show_output() {
    local watch_running=0
    if [[ -n "$shellux_watch_pid" ]] && kill -0 "$shellux_watch_pid" 2>/dev/null; then
      watch_running=1
    elif (( shellux_output_mode )); then
      watch_running=paused
    elif (( shellux_watch_before_command )); then
      watch_running=1
    fi
    if (( ! shellux_enabled )); then
      SHELLUX_DIAG_SHELL=zsh SHELLUX_DIAG_ENABLED=0 SHELLUX_DIAG_WATCH=0 command "${SHELLUX_BIN:-shellux}" "$@"
      return $?
    fi
    shellux_stop_watch || return $?
    # A terminal scroll region cannot preserve lines that scroll past its top
    # edge. Clear the fixed view before using normal terminal scrollback.
    printf '\033[r\033[H\033[2J'
    shellux_render_header=0
    shellux_preserve_content=0
    shellux_output_mode=1
    shellux_command_paused=1
    SHELLUX_DIAG_SHELL=zsh SHELLUX_DIAG_ENABLED=1 SHELLUX_DIAG_WATCH="$watch_running" command "${SHELLUX_BIN:-shellux}" "$@"
  }
  shellux_settings() {
    shellux_stop_watch || return $?
    shellux_render_header=0
    shellux_preserve_content=0
    shellux_output_mode=1
    shellux_command_paused=1
    local settings_result=0
    command "${SHELLUX_BIN:-shellux}" "$@" || settings_result=$?
    case "$settings_result" in
      0)
        if (( shellux_enabled )); then shellux_reload; fi
        ;;
      1) return 0 ;; # Deliberate cancellation is a normal shell completion.
      *) return "$settings_result" ;;
    esac
  }
  shellux() {
    case "${1:-}" in
      "") shellux_reload ;;
      settings) shellux_settings "$@" ;;
      show|background)
        if (( $# == 1 )); then
          shellux_show_output "$@"
        else
          command "${SHELLUX_BIN:-shellux}" "$@" || return $?
          if (( shellux_enabled )); then shellux_reload; fi
        fi
        ;;
      menu)
        if (( $# == 1 )) || [[ "${2:-}" == list ]]; then
          shellux_show_output "$@"
        else
          command "${SHELLUX_BIN:-shellux}" "$@" || return $?
          if (( shellux_enabled )); then shellux_reload; fi
        fi
        ;;
      theme|style|animation)
        command "${SHELLUX_BIN:-shellux}" "$@" || return $?
        if (( shellux_enabled )); then shellux_reload; fi
        ;;
      help|themes|styles|animations|status|doctor)
        shellux_show_output "$@"
        ;;
      off)
        if (( $# != 1 )); then
          command "${SHELLUX_BIN:-shellux}" "$@"
          return $?
        fi
        shellux_off
        ;;
      on)
        if (( $# != 1 )); then
          command "${SHELLUX_BIN:-shellux}" "$@"
          return $?
        fi
        shellux_on
        ;;
      reload) shellux_reload ;;
      exit) exit ;;
      *) command "${SHELLUX_BIN:-shellux}" "$@" ;;
    esac
  }
  if (( $+functions[clear] == 0 )); then
    clear() {
      command clear "$@"
      shellux_render_header=0
      shellux_preserve_content=0
    }
  fi
  shellux_before_command() {
    [[ -t 1 ]] || return 0
    (( shellux_enabled )) || return 0
    (( shellux_command_paused )) && return 0
    shellux_set_title
    shellux_watch_before_command=0
    if [[ -n "$shellux_watch_pid" ]] && kill -0 "$shellux_watch_pid" 2>/dev/null; then
      shellux_watch_before_command=1
    fi
    shellux_stop_watch || return $?
    shellux_output_mode=1
    # The stopped renderer removes its rows before releasing the terminal.
    # Keep command output visible until the user reloads the header.
    # DECSTBM homes the cursor; keep command output at the shell's current row.
    printf '\0337\033[r\0338'
    shellux_render_header=0
    shellux_preserve_content=0
    shellux_command_paused=1
  }
  shellux_handle_resize() {
    [[ -t 1 ]] || return 0
    (( shellux_enabled )) || return 0
    # Help output and foreground applications own the viewport while paused.
    (( shellux_output_mode || shellux_command_paused )) && return 0
    # Terminal.app may reflow its existing screen grid before sending WINCH.
    # Clear only the visible viewport, preserve the cursor, then ask the live
    # renderer to draw one fresh fixed header. Scrollback remains intact.
    printf '\0337\033[H\033[2J\0338'
    if [[ -n "$shellux_watch_pid" ]] && kill -0 "$shellux_watch_pid" 2>/dev/null; then
      kill -WINCH "$shellux_watch_pid" 2>/dev/null || true
    else
      shellux_render_header=1
    fi
    zle reset-prompt 2>/dev/null || true
  }
  autoload -Uz add-zsh-hook
  add-zsh-hook -d precmd shellux_prompt 2>/dev/null || true
  add-zsh-hook -d preexec shellux_mark_clear 2>/dev/null || true
  add-zsh-hook -d preexec shellux_before_command 2>/dev/null || true
  add-zsh-hook precmd shellux_prompt
  add-zsh-hook preexec shellux_before_command
  if (( $+functions[TRAPWINCH] == 0 )); then
    TRAPWINCH() { shellux_handle_resize }
  fi
fi

alias shx='shellux reload'

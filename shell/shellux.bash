# Shellux integration for Bash.
# Source this file from .bashrc after installing the shellux binary.

# Define the alias before DEBUG is installed, so startup does not pause itself.
alias shx='shellux reload'

if ! declare -F shellux_prompt >/dev/null 2>&1 || (( ${shellux_integration_version:-0} < 19 )); then
  shellux_integration_version=19
  : "${shellux_enabled:=1}"
  : "${shellux_render_header:=1}"
  : "${shellux_clear_startup_scrollback:=1}"
  : "${shellux_preserve_content:=0}"
  : "${shellux_output_mode:=0}"
  : "${shellux_command_paused:=0}"
  : "${shellux_watch_pid:=}"
  shellux_watch_before_command=0

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
    command "${SHELLUX_BIN:-shellux}" --watch --in-place --parent-pid $$ </dev/null >/dev/tty 2>/dev/null &
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
    type -P "${SHELLUX_BIN:-shellux}" >/dev/null 2>&1 || return 0
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
      wait "$shellux_watch_pid" 2>/dev/null || true
    fi
    shellux_watch_pid=""
  }
  shellux_reload() {
    if (( ! shellux_enabled )); then
      printf '%s\n' "shellux: header is off; run 'shellux on'" >&2
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
      SHELLUX_DIAG_SHELL=bash SHELLUX_DIAG_ENABLED=0 SHELLUX_DIAG_WATCH=0 command "${SHELLUX_BIN:-shellux}" "$@"
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
    SHELLUX_DIAG_SHELL=bash SHELLUX_DIAG_ENABLED=1 SHELLUX_DIAG_WATCH="$watch_running" command "${SHELLUX_BIN:-shellux}" "$@"
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
  if ! declare -F clear >/dev/null 2>&1; then
    clear() {
      command clear "$@"
      shellux_render_header=0
      shellux_preserve_content=0
    }
  fi
  shellux_before_command() {
    # Bash runs DEBUG before PROMPT_COMMAND as well as user commands.
    [[ "${BASH_COMMAND:-}" == shellux_prompt ]] && return 0
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
    # Rebuild the visible header after the terminal has reflowed its grid.
    # This clears the viewport but leaves the terminal scrollback untouched.
    printf '\0337\033[H\033[2J\0338'
    if [[ -n "$shellux_watch_pid" ]] && kill -0 "$shellux_watch_pid" 2>/dev/null; then
      kill -WINCH "$shellux_watch_pid" 2>/dev/null || true
    else
      shellux_render_header=1
    fi
  }
  if [[ -z "$(trap -p WINCH)" ]]; then
    trap 'shellux_handle_resize' WINCH
  fi
  case ";${PROMPT_COMMAND[*]}" in
    *";shellux_prompt"*) ;;
    *) PROMPT_COMMAND="shellux_prompt${PROMPT_COMMAND:+;$PROMPT_COMMAND}" ;;
  esac
  if [[ -z "$(trap -p DEBUG)" ]]; then
    trap 'shellux_before_command' DEBUG
  fi
fi

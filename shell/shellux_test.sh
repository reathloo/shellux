#!/bin/sh
set -eu

# Exercise both sourced integrations without touching the installed binary or
# user configuration. Every assertion must fail the test, not just the last one.
for test_shell in zsh bash; do
  command -v "$test_shell" >/dev/null 2>&1 || continue
  "$test_shell" -n "shell/shellux.$test_shell"
  SHELLUX_BIN=true "$test_shell" -f -s -- "$test_shell" <<'TESTS'
source "shell/shellux.$1"
set -e

fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }
equal() { [[ "$1" == "$2" ]] || fail "$3: got $1, want $2"; }

equal "$(shellux_set_title)" '' 'window title does not leak into captured output'

shellux_preserve_content=1
TERM=xterm clear >/dev/null
equal "$shellux_preserve_content" 0 'clear does not restore output under a header'
equal "$shellux_render_header" 0 'clear keeps the header paused'

if [[ "$1" == bash ]]; then
  shopt -s expand_aliases
  foreign_debug_hits=0
  foreign_debug_hook() { foreign_debug_hits=$((foreign_debug_hits + 1)); }
  foreign_prompt_one() { :; }
  foreign_prompt_two() { :; }
  trap 'foreign_debug_hook' DEBUG
  PROMPT_COMMAND=(foreign_prompt_one foreign_prompt_two)
  shellux_integration_version=19
  source "shell/shellux.bash"
  equal "$shellux_integration_version" 20 'existing Bash integration upgrades'
  eval "${PROMPT_COMMAND[0]}"
  equal "$shellux_previous_debug_trap" foreign_debug_hook 'existing Bash DEBUG hook is preserved'
  case "$(trap -p DEBUG)" in
    *shellux_debug_dispatch*) ;;
    *) fail 'Bash DEBUG hooks were not chained' ;;
  esac
  case "${PROMPT_COMMAND[0]}" in
    *shellux_prompt*) ;;
    *) fail 'Shellux keeps first prompt hook position' ;;
  esac
  equal "${PROMPT_COMMAND[1]}" foreign_prompt_one 'first existing prompt hook is preserved'
  equal "${PROMPT_COMMAND[2]}" foreign_prompt_two 'second existing prompt hook is preserved'
  debug_hits_before=$foreign_debug_hits
  :
  (( foreign_debug_hits > debug_hits_before )) || fail 'existing Bash DEBUG hook no longer runs'
fi
if [[ "$1" == zsh ]]; then
  foreign_preexec_hook() { :; }
  foreign_precmd_hook() { :; }
  add-zsh-hook preexec foreign_preexec_hook
  add-zsh-hook precmd foreign_precmd_hook
  shellux_integration_version=19
  source "shell/shellux.zsh"
  equal "$shellux_integration_version" 20 'existing Zsh integration upgrades'
  [[ " ${preexec_functions[*]} " == *' foreign_preexec_hook '* ]] || fail 'existing Zsh preexec hook was removed'
  [[ " ${precmd_functions[*]} " == *' foreign_precmd_hook '* ]] || fail 'existing Zsh precmd hook was removed'
fi

# Use a counter instead of launching a renderer; termination is tested below.
watch_starts=0
shellux_start_watch() { watch_starts=$((watch_starts + 1)); }

shellux_output_mode=1
shellux_render_header=1
shellux_prompt >/dev/null
shellux_prompt >/dev/null
equal "$shellux_render_header" 0 'paused output suppresses even a pending redraw'
equal "$shellux_output_mode" 1 'prompt keeps full-window output mode'
equal "$watch_starts" 0 'prompt does not restart animation over output'
shellux reload >/dev/null
equal "$shellux_output_mode" 0 'explicit reload leaves output mode'
equal "$watch_starts" 1 'explicit reload restarts animation'
shellux_output_mode=1
shx >/dev/null
equal "$shellux_output_mode" 0 'shx leaves output mode'
equal "$watch_starts" 2 'shx reloads exactly once'
watch_starts=0

shellux off >/dev/null
shellux off >/dev/null
equal "$shellux_enabled" 0 'off is idempotent'
if shellux reload >/dev/null 2>&1; then fail 'reload accepted disabled header'; fi
shellux on >/dev/null
shellux on >/dev/null
equal "$shellux_enabled" 1 'on enables header'
equal "$watch_starts" 1 'on is idempotent'

for subcommand in theme style animation; do
  shellux "$subcommand" purplemonster >/dev/null
done

equal "$watch_starts" 4 'appearance commands reload exactly once'
shellux >/dev/null
equal "$watch_starts" 5 'bare shellux reloads exactly once'

for subcommand in help themes styles animations status doctor; do
  shellux "$subcommand" >/dev/null
  equal "$shellux_output_mode" 1 'informational command uses scrollback'
  equal "$shellux_render_header" 0 'informational output stays visible'
done
equal "$watch_starts" 5 'informational commands do not start a renderer'

watch_starts=0
shellux menu >/dev/null
shellux menu list >/dev/null
equal "$watch_starts" 0 'menu lists do not reload'
shellux show set cpu ram >/dev/null
shellux show replace cpu uptime >/dev/null
shellux menu reset >/dev/null
equal "$watch_starts" 3 'selection changes reload exactly once'
shellux off >/dev/null
shellux show spotify off >/dev/null
equal "$watch_starts" 3 'selection changes while off do not start renderer'
shellux on >/dev/null
SHELLUX_BIN=false
watch_starts=0
if shellux show replace cpu uptime >/dev/null; then fail 'failed selection change reported success'; fi
equal "$watch_starts" 0 'failed selection change does not reload'
SHELLUX_BIN=true

watch_starts=0
shellux show >/dev/null
equal "$watch_starts" 0 'show list does not reload'
equal "$shellux_output_mode" 1 'show list stays readable'
shellux show cpu off >/dev/null
shellux show cpu bar off >/dev/null
equal "$watch_starts" 2 'show changes reload exactly once'
shellux off >/dev/null
shellux show ram percent off >/dev/null
equal "$watch_starts" 2 'show while off does not start renderer'
shellux on >/dev/null
watch_starts=0
SHELLUX_BIN=false
if shellux show cpu off >/dev/null; then fail 'failed show reported success'; fi
equal "$watch_starts" 0 'failed show does not reload'
SHELLUX_BIN=true

watch_starts=0
shellux background >/dev/null
equal "$watch_starts" 0 'background listing does not reload'
equal "$shellux_output_mode" 1 'background listing stays readable'
shellux background dark >/dev/null
shellux background default >/dev/null
equal "$watch_starts" 2 'background changes reload exactly once'
shellux off >/dev/null
shellux background navy >/dev/null
equal "$watch_starts" 2 'background changes while off do not start renderer'
shellux on >/dev/null
watch_starts=0
SHELLUX_BIN=false
if shellux background invalid >/dev/null; then fail 'failed background change reported success'; fi
equal "$watch_starts" 0 'failed background change does not reload'
SHELLUX_BIN=true

# Simulate the UI exit protocol separately from the renderer exit status.
settings_test_bin=$(mktemp)
cat > "$settings_test_bin" <<'SCRIPT'
#!/bin/sh
if [ "$1" = settings ]; then exit "${SHELLUX_SETTINGS_TEST_CODE:-0}"; fi
exit 0
SCRIPT
chmod +x "$settings_test_bin"
SHELLUX_BIN=$settings_test_bin
export SHELLUX_SETTINGS_TEST_CODE=0
watch_starts=0
shellux settings >/dev/null
equal "$watch_starts" 1 'saved settings reload exactly once'
equal "$shellux_output_mode" 0 'saved settings leave output mode'
SHELLUX_SETTINGS_TEST_CODE=1
shellux settings >/dev/null || fail 'intentional cancellation failed'
equal "$watch_starts" 1 'cancelled settings do not reload'
equal "$shellux_output_mode" 1 'cancelled settings keep output readable'
for settings_code in 2 130; do
  SHELLUX_SETTINGS_TEST_CODE=$settings_code
  settings_result=0
  shellux settings >/dev/null || settings_result=$?
  equal "$settings_result" "$settings_code" 'settings failures keep exit code'
  equal "$watch_starts" 1 'failed settings do not reload'
done
shellux off >/dev/null
SHELLUX_SETTINGS_TEST_CODE=0
shellux settings >/dev/null
equal "$shellux_enabled" 0 'settings preserve explicitly disabled tab'
equal "$watch_starts" 1 'saved settings while off do not start renderer'
rm "$settings_test_bin"
unset SHELLUX_SETTINGS_TEST_CODE
SHELLUX_BIN=true
shellux on >/dev/null

sleep 30 &
test_pid=$!
trap 'kill "$test_pid" 2>/dev/null || true' EXIT
shellux_watch_pid=$test_pid
shellux reload >/dev/null
if kill -0 "$test_pid" 2>/dev/null; then fail 'reload left old renderer running'; fi
equal "$shellux_watch_pid" '' 'reload clears old renderer PID'
trap - EXIT

SHELLUX_BIN=false
watch_starts=0
shellux_render_header=1
shellux_clear_startup_scrollback=1
if shellux reload >/dev/null; then fail 'failed reload reported success'; fi
equal "$watch_starts" 0 'failed reload did not start renderer'
equal "$shellux_render_header" 1 'failed reload keeps redraw pending'

if shellux_prompt >/dev/null; then fail 'failed startup render reported success'; fi
equal "$watch_starts" 0 'failed startup did not start renderer'
equal "$shellux_render_header" 1 'failed startup keeps redraw pending'
equal "$shellux_clear_startup_scrollback" 1 'failed startup keeps initial clear pending'

shellux off >/dev/null
if shellux on >/dev/null; then fail 'failed on reported success'; fi
equal "$shellux_enabled" 0 'failed on can be retried'
SHELLUX_BIN=true
shellux on >/dev/null
equal "$shellux_enabled" 1 'on recovers after rendering failure'
equal "$watch_starts" 1 'recovery starts exactly one renderer'

printf '%s integration tests passed\n' "$1"
shellux exit
fail 'shellux exit did not exit'
TESTS
done

# Loading Starship after Shellux still leaves its DEBUG hook in place until the
# first prompt. Shellux then preserves it and yields to it before pausing.
SHELLUX_BIN=true bash --noprofile --norc <<'TESTS'
source shell/shellux.bash
STARSHIP_DEBUG_TRAP=
starship_preexec_all() { eval -- "$STARSHIP_DEBUG_TRAP"; }
trap 'starship_preexec_all' DEBUG
STARSHIP_SHELL=bash
eval "$PROMPT_COMMAND"
case "$(trap -p DEBUG)" in
  *shellux_debug_dispatch*) ;;
  *) printf 'FAIL: Shellux hook was not installed after Starship\n' >&2; exit 1 ;;
esac
[[ "$shellux_previous_debug_trap" == starship_preexec_all ]] || {
  printf 'FAIL: Shellux did not retain the Starship hook\n' >&2
  exit 1
}
TESTS

# bash-preexec exposes cooperative arrays specifically for integrations. Use
# them when present instead of competing for its DEBUG trap.
SHELLUX_BIN=true bash --noprofile --norc <<'TESTS'
bash_preexec_imported=1
preexec_functions=(foreign_preexec)
precmd_functions=(foreign_precmd)
trap 'true' DEBUG
PROMPT_COMMAND=(foreign_precmd)
source shell/shellux.bash
eval "${PROMPT_COMMAND[0]}"
[[ " ${preexec_functions[*]} " == *' shellux_before_command '* ]] || {
  printf 'FAIL: bash-preexec did not receive the Shellux hook\n' >&2
  exit 1
}
[[ "$(trap -p DEBUG)" == "trap -- 'true' DEBUG" ]] || {
  printf 'FAIL: bash-preexec DEBUG trap was replaced\n' >&2
  exit 1
}
TESTS

printf 'compatibility hook tests passed\n'

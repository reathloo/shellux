#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT HUP INT TERM

mkdir -p "$fixture/archive/shell" "$fixture/home"
printf '%s\n' '#!/bin/sh' 'exit 0' > "$fixture/archive/shellux"
cp "$repo_root/scripts/install.sh" "$fixture/archive/install.sh"
cp "$repo_root/shell/shellux.zsh" "$fixture/archive/shell/shellux.zsh"
cp "$repo_root/shell/shellux.bash" "$fixture/archive/shell/shellux.bash"
chmod 0755 "$fixture/archive/shellux" "$fixture/archive/install.sh"

HOME="$fixture/home" \
SHELLUX_PREFIX="$fixture/prefix" \
XDG_DATA_HOME="$fixture/data" \
  "$fixture/archive/install.sh" > "$fixture/output"

test -x "$fixture/prefix/bin/shellux"
test -f "$fixture/data/shellux/shell/shellux.zsh"
test -f "$fixture/data/shellux/shell/shellux.bash"
grep -F "$fixture/prefix/bin/shellux" "$fixture/output" >/dev/null

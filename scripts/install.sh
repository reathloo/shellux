#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
source_binary="$script_dir/shellux"
source_shell="$script_dir/shell"

if [ ! -f "$source_binary" ] || [ ! -f "$source_shell/shellux.zsh" ] || [ ! -f "$source_shell/shellux.bash" ]; then
  printf '%s\n' "shellux: run install.sh from an extracted Shellux release archive" >&2
  exit 1
fi

prefix=${SHELLUX_PREFIX:-"$HOME/.local"}
data_home=${XDG_DATA_HOME:-"$HOME/.local/share"}
integration_dir="$data_home/shellux/shell"

install -d "$prefix/bin" "$integration_dir"
install -m 0755 "$source_binary" "$prefix/bin/shellux"
install -m 0644 "$source_shell/shellux.zsh" "$integration_dir/shellux.zsh"
install -m 0644 "$source_shell/shellux.bash" "$integration_dir/shellux.bash"

cat <<EOF
Shellux was installed to $prefix/bin/shellux

Add these lines to ~/.zshrc:
  export PATH="$prefix/bin:\$PATH"
  source "$integration_dir/shellux.zsh"

For Bash, use ~/.bashrc and source "$integration_dir/shellux.bash" instead.
EOF

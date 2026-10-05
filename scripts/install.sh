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

print_manual_setup() {
  cat <<EOF

Add these lines to your shell configuration:
  export PATH="$prefix/bin:\$PATH"
  source "$integration_dir/shellux.zsh"

For Bash, source "$integration_dir/shellux.bash" instead.
EOF
}

shell_quote() {
  printf "'"
  printf '%s' "$1" | sed "s/'/'\\\\''/g"
  printf "'"
}

sources_integration() {
  awk -v file="shellux.$shell_name" '
    /^[[:space:]]*#/ { next }
    /^[[:space:]]*(source|\.)[[:space:]]/ && index($0, file) { found = 1 }
    END { exit !found }
  ' "$rc_file"
}

sources_bashrc() {
  awk '
    /^[[:space:]]*#/ { next }
    index($0, ".bashrc") && /(^|[;[:space:]])(source|\.)[[:space:]]/ { found = 1 }
    END { exit !found }
  ' "$1"
}

configure_bash_login() {
  for candidate in .bash_profile .bash_login; do
    if [ -f "$HOME/$candidate" ]; then
      login_file="$HOME/$candidate"
      break
    fi
  done
  if [ -z "${login_file:-}" ]; then
    login_file="$HOME/.bash_profile"
    if [ -f "$HOME/.profile" ]; then
      printf '%s\n' '[ -f "$HOME/.profile" ] && . "$HOME/.profile"' >> "$login_file"
      if sources_bashrc "$HOME/.profile"; then
        return
      fi
    fi
  fi
  touch "$login_file"
  if sources_bashrc "$login_file"; then
    return
  fi
  if [ -f "$HOME/.profile" ] && grep -F '. "$HOME/.profile"' "$login_file" >/dev/null 2>&1 && sources_bashrc "$HOME/.profile"; then
    return
  fi
  {
    printf '\n%s\n' '# >>> shellux bash login >>>'
    printf '%s\n' 'if [ -f "$HOME/.bashrc" ]; then'
    printf '%s\n' '  . "$HOME/.bashrc"'
    printf '%s\n' 'fi'
    printf '%s\n' '# <<< shellux bash login <<<'
  } >> "$login_file"
  printf '%s\n' "Shellux made $login_file load .bashrc"
}

configure_shell() {
  current_shell=${SHELL:-}
  shell_name=${current_shell##*/}
  case "$shell_name" in
    zsh)
      rc_file="${ZDOTDIR:-$HOME}/.zshrc"
      integration_file="$integration_dir/shellux.zsh"
      ;;
    bash)
      rc_file="$HOME/.bashrc"
      integration_file="$integration_dir/shellux.bash"
      ;;
    *)
      printf '%s\n' "Shellux could not configure ${current_shell:-the current shell} automatically."
      print_manual_setup
      return
      ;;
  esac

  rc_dir=${rc_file%/*}
  install -d "$rc_dir"
  touch "$rc_file"

  if sources_integration; then
    printf '%s\n' "Shellux is already configured in $rc_file"
  else
    quoted_prefix=$(shell_quote "$prefix")
    quoted_integration=$(shell_quote "$integration_file")
    {
      printf '\n%s\n' '# >>> shellux >>>'
      printf 'export PATH=%s/bin:$PATH\n' "$quoted_prefix"
      printf 'source %s\n' "$quoted_integration"
      printf '%s\n' '# <<< shellux <<<'
    } >> "$rc_file"
    printf '%s\n' "Shellux was added to $rc_file"
  fi
  if [ "$shell_name" = bash ]; then
    configure_bash_login
  fi
}

printf '%s\n' "Shellux was installed to $prefix/bin/shellux"

if [ "${SHELLUX_AUTO_CONFIGURE:-0}" = "1" ]; then
  configure_shell
  printf '%s\n' "Open a new terminal tab to start Shellux."
else
  print_manual_setup
fi

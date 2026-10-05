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

SHELL=/bin/zsh \
HOME="$fixture/home" \
SHELLUX_PREFIX="$fixture/prefix" \
XDG_DATA_HOME="$fixture/data" \
SHELLUX_AUTO_CONFIGURE=1 \
  "$fixture/archive/install.sh" > "$fixture/automatic-output"

grep -F '# >>> shellux >>>' "$fixture/home/.zshrc" >/dev/null
grep -F "$fixture/data/shellux/shell/shellux.zsh" "$fixture/home/.zshrc" >/dev/null
grep -F "Shellux was added to $fixture/home/.zshrc" "$fixture/automatic-output" >/dev/null

SHELL=/bin/zsh \
HOME="$fixture/home" \
SHELLUX_PREFIX="$fixture/prefix" \
XDG_DATA_HOME="$fixture/data" \
SHELLUX_AUTO_CONFIGURE=1 \
  "$fixture/archive/install.sh" > "$fixture/repeated-output"

test "$(grep -c '# >>> shellux >>>' "$fixture/home/.zshrc")" -eq 1
grep -F "Shellux is already configured in $fixture/home/.zshrc" "$fixture/repeated-output" >/dev/null

mkdir "$fixture/legacy-home"
printf '%s\n' 'export PATH="$HOME/.local/bin:$PATH"' 'source "$HOME/.local/share/shellux/shell/shellux.zsh"' > "$fixture/legacy-home/.zshrc"
SHELL=/bin/zsh \
HOME="$fixture/legacy-home" \
SHELLUX_AUTO_CONFIGURE=1 \
  "$fixture/archive/install.sh" > "$fixture/legacy-output"
test "$(grep -c 'shellux.zsh' "$fixture/legacy-home/.zshrc")" -eq 1
test "$(grep -c '# >>> shellux >>>' "$fixture/legacy-home/.zshrc" || true)" -eq 0
grep -F "Shellux is already configured in $fixture/legacy-home/.zshrc" "$fixture/legacy-output" >/dev/null

release_dir="$fixture/release"
mkdir -p "$release_dir"
case $(uname -s) in
  Darwin) release_os=darwin ;;
  Linux) release_os=linux ;;
  *) exit 1 ;;
esac
case $(uname -m) in
  x86_64|amd64) release_arch=amd64 ;;
  arm64|aarch64) release_arch=arm64 ;;
  *) exit 1 ;;
esac

archive_name="shellux_0.1.2_${release_os}_${release_arch}.tar.gz"
tar -czf "$release_dir/$archive_name" -C "$fixture/archive" .
if command -v sha256sum >/dev/null 2>&1; then
  archive_checksum=$(sha256sum "$release_dir/$archive_name" | awk '{ print $1 }')
else
  archive_checksum=$(shasum -a 256 "$release_dir/$archive_name" | awk '{ print $1 }')
fi
printf '%s  %s\n' "$archive_checksum" "$archive_name" > "$release_dir/checksums.txt"

mkdir "$fixture/bootstrap-home"
SHELL=/bin/bash \
HOME="$fixture/bootstrap-home" \
SHELLUX_PREFIX="$fixture/bootstrap-prefix" \
XDG_DATA_HOME="$fixture/bootstrap-data" \
SHELLUX_RELEASE_BASE_URL="file://$release_dir" \
  sh "$repo_root/scripts/install-release.sh" > "$fixture/bootstrap-output"

test -x "$fixture/bootstrap-prefix/bin/shellux"
test -f "$fixture/bootstrap-data/shellux/shell/shellux.bash"
grep -F '# >>> shellux >>>' "$fixture/bootstrap-home/.bashrc" >/dev/null
grep -F '. "$HOME/.bashrc"' "$fixture/bootstrap-home/.bash_profile" >/dev/null
grep -F "Verified $archive_name" "$fixture/bootstrap-output" >/dev/null

SHELL=/bin/bash \
HOME="$fixture/bootstrap-home" \
SHELLUX_PREFIX="$fixture/bootstrap-prefix" \
XDG_DATA_HOME="$fixture/bootstrap-data" \
SHELLUX_AUTO_CONFIGURE=1 \
  "$fixture/archive/install.sh" > "$fixture/bash-repeated-output"
test "$(grep -c '# >>> shellux >>>' "$fixture/bootstrap-home/.bashrc")" -eq 1
test "$(grep -c '# >>> shellux bash login >>>' "$fixture/bootstrap-home/.bash_profile")" -eq 1
HOME="$fixture/bootstrap-home" SHELLUX_BIN=true bash --noprofile --norc -c '. "$HOME/.bash_profile"; declare -F shellux >/dev/null'

mkdir "$fixture/existing-login-home"
printf '%s\n' 'if [ -f "$HOME/.bashrc" ]; then . "$HOME/.bashrc"; fi' > "$fixture/existing-login-home/.bash_profile"
SHELL=/bin/bash \
HOME="$fixture/existing-login-home" \
SHELLUX_PREFIX="$fixture/existing-login-prefix" \
XDG_DATA_HOME="$fixture/existing-login-data" \
SHELLUX_AUTO_CONFIGURE=1 \
  "$fixture/archive/install.sh" > "$fixture/existing-login-output"
test "$(grep -c '# >>> shellux bash login >>>' "$fixture/existing-login-home/.bash_profile" || true)" -eq 0

mkdir "$fixture/profile-home"
printf '%s\n' 'export SHELLUX_PROFILE_MARKER=present' > "$fixture/profile-home/.profile"
SHELL=/bin/bash \
HOME="$fixture/profile-home" \
SHELLUX_PREFIX="$fixture/profile-prefix" \
XDG_DATA_HOME="$fixture/profile-data" \
SHELLUX_AUTO_CONFIGURE=1 \
  "$fixture/archive/install.sh" > "$fixture/profile-output"
test "$(grep -c 'SHELLUX_PROFILE_MARKER' "$fixture/profile-home/.profile")" -eq 1
grep -F '. "$HOME/.profile"' "$fixture/profile-home/.bash_profile" >/dev/null
HOME="$fixture/profile-home" SHELLUX_BIN=true bash --noprofile --norc -c '. "$HOME/.bash_profile"; test "$SHELLUX_PROFILE_MARKER" = present; declare -F shellux >/dev/null'

mkdir "$fixture/profile-sources-home"
printf '%s\n' '[ -f "$HOME/.bashrc" ] && . "$HOME/.bashrc"' > "$fixture/profile-sources-home/.profile"
SHELL=/bin/bash \
HOME="$fixture/profile-sources-home" \
SHELLUX_PREFIX="$fixture/profile-sources-prefix" \
XDG_DATA_HOME="$fixture/profile-sources-data" \
SHELLUX_AUTO_CONFIGURE=1 \
  "$fixture/archive/install.sh" > "$fixture/profile-sources-output"
test "$(grep -c '.bashrc' "$fixture/profile-sources-home/.bash_profile" || true)" -eq 0
HOME="$fixture/profile-sources-home" SHELLUX_BIN=true bash --noprofile --norc -c '. "$HOME/.bash_profile"; declare -F shellux >/dev/null'
SHELL=/bin/bash \
HOME="$fixture/profile-sources-home" \
SHELLUX_PREFIX="$fixture/profile-sources-prefix" \
XDG_DATA_HOME="$fixture/profile-sources-data" \
SHELLUX_AUTO_CONFIGURE=1 \
  "$fixture/archive/install.sh" > "$fixture/profile-sources-repeated-output"
test "$(grep -c '.bashrc' "$fixture/profile-sources-home/.bash_profile" || true)" -eq 0

printf '%064d  %s\n' 0 "$archive_name" > "$release_dir/checksums.txt"
if SHELL=/bin/bash \
  HOME="$fixture/bootstrap-home" \
  SHELLUX_PREFIX="$fixture/rejected-prefix" \
  XDG_DATA_HOME="$fixture/rejected-data" \
  SHELLUX_RELEASE_BASE_URL="file://$release_dir" \
    sh "$repo_root/scripts/install-release.sh" > "$fixture/rejected-output" 2>&1; then
  printf '%s\n' 'installer accepted an archive with the wrong checksum' >&2
  exit 1
fi
test ! -e "$fixture/rejected-prefix/bin/shellux"
grep -F 'checksum verification failed' "$fixture/rejected-output" >/dev/null

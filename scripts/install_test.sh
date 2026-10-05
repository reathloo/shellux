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

archive_name="shellux_0.1.1_${release_os}_${release_arch}.tar.gz"
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
grep -F "Verified $archive_name" "$fixture/bootstrap-output" >/dev/null

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

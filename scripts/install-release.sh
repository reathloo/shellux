#!/bin/sh
set -eu

repository=${SHELLUX_RELEASE_BASE_URL:-"https://github.com/reathloo/shellux/releases/latest/download"}
temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/shellux-install.XXXXXX")
trap 'rm -rf "$temporary_dir"' EXIT HUP INT TERM

for command_name in curl tar; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    printf '%s\n' "shellux: required command not found: $command_name" >&2
    exit 1
  fi
done

case $(uname -s) in
  Darwin) operating_system=darwin ;;
  Linux) operating_system=linux ;;
  *)
    printf '%s\n' "shellux: this installer supports macOS and Linux" >&2
    exit 1
    ;;
esac

case $(uname -m) in
  x86_64|amd64) architecture=amd64 ;;
  arm64|aarch64) architecture=arm64 ;;
  *)
    printf '%s\n' "shellux: unsupported processor architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

checksums="$temporary_dir/checksums.txt"
curl -fsSL --retry 3 --retry-delay 1 "$repository/checksums.txt" -o "$checksums"

suffix="_${operating_system}_${architecture}.tar.gz"
archive_name=$(awk -v suffix="$suffix" 'index($2, suffix) == length($2) - length(suffix) + 1 { print $2 }' "$checksums")
archive_count=$(printf '%s\n' "$archive_name" | awk 'NF { count++ } END { print count + 0 }')
if [ "$archive_count" -ne 1 ]; then
  printf '%s\n' "shellux: could not identify one release archive for ${operating_system}/${architecture}" >&2
  exit 1
fi

case "$archive_name" in
  */*|*..*)
    printf '%s\n' "shellux: invalid archive name in checksums.txt" >&2
    exit 1
    ;;
esac

expected_checksum=$(awk -v archive="$archive_name" '$2 == archive { print $1 }' "$checksums")
archive="$temporary_dir/$archive_name"
curl -fsSL --retry 3 --retry-delay 1 "$repository/$archive_name" -o "$archive"

if command -v sha256sum >/dev/null 2>&1; then
  actual_checksum=$(sha256sum "$archive" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
  actual_checksum=$(shasum -a 256 "$archive" | awk '{ print $1 }')
else
  printf '%s\n' "shellux: sha256sum or shasum is required to verify the download" >&2
  exit 1
fi

if [ "$actual_checksum" != "$expected_checksum" ]; then
  printf '%s\n' "shellux: checksum verification failed" >&2
  exit 1
fi

members="$temporary_dir/archive-members.txt"
tar -tzf "$archive" > "$members"
if awk '
  /^\// || /^\.\.\// || /\/\.\.\// || /\/\.\.$/ || $0 == ".." { unsafe = 1 }
  END { exit unsafe ? 0 : 1 }
' "$members"; then
  printf '%s\n' "shellux: archive contains an unsafe path" >&2
  exit 1
fi

find_member() {
  awk -v wanted="$1" '
    {
      normalized = $0
      while (substr(normalized, 1, 2) == "./") {
        normalized = substr(normalized, 3)
      }
      if (normalized == wanted) {
        member = $0
        count++
      }
    }
    END {
      if (count != 1) exit 1
      print member
    }
  ' "$members"
}

if ! binary_member=$(find_member shellux) ||
   ! installer_member=$(find_member install.sh) ||
   ! zsh_member=$(find_member shell/shellux.zsh) ||
   ! bash_member=$(find_member shell/shellux.bash); then
  printf '%s\n' "shellux: release archive is missing or duplicates a required file" >&2
  exit 1
fi

extract_dir="$temporary_dir/release"
mkdir -p "$extract_dir/shell"
tar -xOzf "$archive" "$binary_member" > "$extract_dir/shellux"
tar -xOzf "$archive" "$installer_member" > "$extract_dir/install.sh"
tar -xOzf "$archive" "$zsh_member" > "$extract_dir/shell/shellux.zsh"
tar -xOzf "$archive" "$bash_member" > "$extract_dir/shell/shellux.bash"

for required_file in shellux install.sh shell/shellux.zsh shell/shellux.bash; do
  if [ ! -s "$extract_dir/$required_file" ]; then
    printf '%s\n' "shellux: release archive contains an invalid $required_file" >&2
    exit 1
  fi
done
chmod 0755 "$extract_dir/shellux" "$extract_dir/install.sh"
chmod 0644 "$extract_dir/shell/shellux.zsh" "$extract_dir/shell/shellux.bash"

printf '%s\n' "Verified $archive_name"
SHELLUX_AUTO_CONFIGURE=1 "$extract_dir/install.sh"

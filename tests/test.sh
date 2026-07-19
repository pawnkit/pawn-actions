#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary="$(mktemp -d)"
trap 'rm -rf "$temporary"' EXIT

digest() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

mkdir -p "$temporary/archive" "$temporary/bin" "$temporary/install"
cat > "$temporary/archive/pawn" <<'SCRIPT'
#!/usr/bin/env bash
echo "pawn 1.2.3"
SCRIPT
chmod +x "$temporary/archive/pawn"
tar -czf "$temporary/pawn.tar.gz" -C "$temporary/archive" pawn
checksum="$(digest "$temporary/pawn.tar.gz")"

cat > "$temporary/bin/curl" <<'SCRIPT'
#!/usr/bin/env bash
set -euo pipefail
destination=''
while (($#)); do
  if [[ "$1" == "--output" ]]; then destination="$2"; shift 2; else shift; fi
done
cp "$FIXTURE_ARCHIVE" "$destination"
SCRIPT
chmod +x "$temporary/bin/curl"

export PATH="$temporary/bin:$PATH"
export FIXTURE_ARCHIVE="$temporary/pawn.tar.gz"
export PAWN_VERSION="1.2.3"
export PAWN_DOWNLOAD_URL="https://example.test/pawn.tar.gz"
export PAWN_SHA256="$checksum"
export PAWN_CACHE_HIT="false"
export PAWN_INSTALL_DIR="$temporary/install"
export GITHUB_PATH="$temporary/github-path"
export GITHUB_OUTPUT="$temporary/github-output"
export RUNNER_OS="Linux"

export PAWN_DOWNLOAD_URL="https://user:secret@example.test/pawn.tar.gz"
if "$root/setup/install.sh" >/dev/null 2>&1; then
  echo "credential-bearing URL was accepted" >&2
  exit 1
fi
export PAWN_DOWNLOAD_URL="https://example.test/pawn.tar.gz"

"$root/setup/install.sh"

export PAWN_VERSION="latest"
if "$root/setup/install.sh" >/dev/null 2>&1; then
  echo "non-version setup input was accepted" >&2
  exit 1
fi
export PAWN_VERSION="1.2.3"
test -x "$temporary/install/pawn"
test "$(cat "$temporary/install/.archive-sha256")" = "$checksum"

cat > "$temporary/bin/curl" <<'SCRIPT'
#!/usr/bin/env bash
exit 99
SCRIPT
export PAWN_CACHE_HIT="true"
"$root/setup/install.sh"

go run "$root/tests/archivefixture" "$temporary/archive/pawn" "$temporary/unsafe.tar.gz"
cat > "$temporary/bin/curl" <<'SCRIPT'
#!/usr/bin/env bash
set -euo pipefail
destination=''
while (($#)); do
  if [[ "$1" == "--output" ]]; then destination="$2"; shift 2; else shift; fi
done
cp "$FIXTURE_ARCHIVE" "$destination"
SCRIPT
chmod +x "$temporary/bin/curl"

mkdir "$temporary/windows"
cat > "$temporary/windows/pawn.exe" <<'SCRIPT'
#!/usr/bin/env bash
echo "pawn 1.2.3"
SCRIPT
chmod +x "$temporary/windows/pawn.exe"
tar -czf "$temporary/windows.tar.gz" -C "$temporary/windows" pawn.exe
export FIXTURE_ARCHIVE="$temporary/windows.tar.gz"
export PAWN_CACHE_HIT="false"
export PAWN_INSTALL_DIR="$temporary/windows-install"
export PAWN_SHA256="$(digest "$temporary/windows.tar.gz")"
export RUNNER_OS="Windows"
"$root/setup/install.sh"
test -x "$temporary/windows-install/pawn.exe"

export RUNNER_OS="Linux"
export FIXTURE_ARCHIVE="$temporary/unsafe.tar.gz"
export PAWN_CACHE_HIT="false"
export PAWN_INSTALL_DIR="$temporary/unsafe-install"
export PAWN_SHA256="$(digest "$temporary/unsafe.tar.gz")"
if "$root/setup/install.sh" >/dev/null 2>&1; then
  echo "unsafe archive path was accepted" >&2
  exit 1
fi

mkdir "$temporary/symlink"
if ln -s /tmp "$temporary/symlink/pawn" 2>/dev/null && [[ -L "$temporary/symlink/pawn" ]]; then
  tar -czf "$temporary/symlink.tar.gz" -C "$temporary/symlink" pawn
  export FIXTURE_ARCHIVE="$temporary/symlink.tar.gz"
  export PAWN_INSTALL_DIR="$temporary/symlink-install"
  export PAWN_SHA256="$(digest "$temporary/symlink.tar.gz")"
  if "$root/setup/install.sh" >/dev/null 2>&1; then
    echo "symlink binary was accepted" >&2
    exit 1
  fi
fi

export PAWN_CACHE_HIT="false"
export PAWN_SHA256="$(printf '0%.0s' {1..64})"
if "$root/setup/install.sh" >/dev/null 2>&1; then
  echo "checksum mismatch was accepted" >&2
  exit 1
fi

bash -n "$root/setup/validate.sh" "$root/setup/install.sh" "$root/check/run.sh"

while IFS= read -r use; do
  target="${use#*uses: }"
  if [[ "$target" == pawnkit/* ]]; then continue; fi
  reference="${target##*@}"
  if [[ ! "$reference" =~ ^[0-9a-f]{40}$ ]]; then
    echo "third-party action is not pinned: $target" >&2
    exit 1
  fi
done < <(grep -RhoE 'uses: [^[:space:]]+' "$root/.github" "$root/setup" "$root/check" "$root/fmt" "$root/lint" "$root/test")

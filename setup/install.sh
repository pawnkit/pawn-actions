#!/usr/bin/env bash
set -euo pipefail

"$(dirname "${BASH_SOURCE[0]}")/validate.sh"

binary="$PAWN_INSTALL_DIR/pawn"
archive_binary="pawn"
if [[ "${RUNNER_OS:-}" == "Windows" ]]; then
  binary="$PAWN_INSTALL_DIR/pawn.exe"
  archive_binary="pawn.exe"
fi
marker="$PAWN_INSTALL_DIR/.archive-sha256"
cache_valid=false
if [[ "$PAWN_CACHE_HIT" == "true" && -x "$binary" && -f "$marker" ]]; then
  expected_version="pawn ${PAWN_VERSION#v}"
  if [[ "$(tr -d '\r\n' < "$marker")" == "$PAWN_SHA256" && "$("$binary" version 2>/dev/null)" == "$expected_version" ]]; then
    cache_valid=true
  fi
fi

if [[ "$cache_valid" != "true" ]]; then
  temporary="$(mktemp -d)"
  trap 'rm -rf "$temporary"' EXIT
  archive="$temporary/pawn.tar.gz"
  curl --fail --location --proto '=https' --proto-redir '=https' --tlsv1.2 --output "$archive" "$PAWN_DOWNLOAD_URL"
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$archive" | awk '{print $1}')"
  else
    actual="$(shasum -a 256 "$archive" | awk '{print $1}')"
  fi
  if [[ "$actual" != "$PAWN_SHA256" ]]; then
    echo "setup-pawn: checksum mismatch" >&2
    exit 1
  fi
  entry_count=0
  archive_entry=""
  while IFS= read -r entry; do
    entry_count=$((entry_count + 1))
    archive_entry="${entry#./}"
  done < <(tar -tzf "$archive")
  if [[ "$entry_count" -ne 1 || "$archive_entry" != "$archive_binary" ]]; then
    echo "setup-pawn: archive must contain only $archive_binary at its root" >&2
    exit 1
  fi
  staging="$temporary/extract"
  mkdir -p "$staging"
  tar -xzf "$archive" -C "$staging"
  staged_binary="$staging/$archive_binary"
  if [[ ! -f "$staged_binary" || -L "$staged_binary" ]]; then
    echo "setup-pawn: archive does not contain a regular $archive_binary" >&2
    exit 1
  fi
  mkdir -p "$PAWN_INSTALL_DIR"
  cp "$staged_binary" "$binary"
  chmod +x "$binary"
  expected_version="pawn ${PAWN_VERSION#v}"
  if [[ "$("$binary" version 2>/dev/null)" != "$expected_version" ]]; then
    echo "setup-pawn: binary version does not match $PAWN_VERSION" >&2
    exit 1
  fi
  printf '%s\n' "$PAWN_SHA256" > "$marker"
fi

"$binary" version
echo "$PAWN_INSTALL_DIR" >> "$GITHUB_PATH"
echo "version=$PAWN_VERSION" >> "$GITHUB_OUTPUT"
echo "cache-hit=$cache_valid" >> "$GITHUB_OUTPUT"
echo "path=$PAWN_INSTALL_DIR" >> "$GITHUB_OUTPUT"

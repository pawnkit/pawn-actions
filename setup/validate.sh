#!/usr/bin/env bash
set -euo pipefail

if [[ ! "$PAWN_VERSION" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?(\+[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$ ]]; then
  echo "setup-pawn: version must be an exact semantic version" >&2
  exit 2
fi
if [[ ! "$PAWN_SHA256" =~ ^[0-9a-f]{64}$ ]]; then
  echo "setup-pawn: sha256 must be 64 lowercase hexadecimal characters" >&2
  exit 2
fi
if [[ "$PAWN_DOWNLOAD_URL" != https://* ]]; then
  echo "setup-pawn: download-url must use HTTPS" >&2
  exit 2
fi
authority="${PAWN_DOWNLOAD_URL#https://}"
authority="${authority%%/*}"
if [[ "$authority" == *"@"* ]]; then
  echo "setup-pawn: download-url must not contain credentials" >&2
  exit 2
fi

#!/usr/bin/env bash
set -euo pipefail

if [[ -z "$PAWN_COMPILER" && -z "$PAWN_BACKEND" ]]; then
  echo "compiler or backend is required" >&2
  exit 2
fi
if [[ -n "$PAWN_COMPILER" && -n "$PAWN_BACKEND" ]]; then
  echo "compiler and backend cannot be used together" >&2
  exit 2
fi

args=(build --project "$PAWN_PROJECT" --format "$PAWN_OUTPUT")
if [[ -n "$PAWN_PROFILE" ]]; then args+=(--profile "$PAWN_PROFILE"); fi
if [[ -n "$PAWN_BUILD" ]]; then args+=(--build "$PAWN_BUILD"); fi
if [[ -n "$PAWN_RUNTIME" ]]; then args+=(--runtime "$PAWN_RUNTIME"); fi
if [[ -n "$PAWN_COMPILER" ]]; then args+=(--compiler "$PAWN_COMPILER"); fi
if [[ -n "$PAWN_BACKEND" ]]; then args+=(--backend "$PAWN_BACKEND"); fi
if [[ -n "$PAWN_ARTIFACT" ]]; then args+=(--artifact "$PAWN_ARTIFACT"); fi

if [[ -n "$PAWN_RESULT_FILE" ]]; then
  mkdir -p "$(dirname "$PAWN_RESULT_FILE")"
  set +e
  pawn "${args[@]}" > "$PAWN_RESULT_FILE"
  status=$?
  set -e
  if ((status != 0)); then
    cat "$PAWN_RESULT_FILE" >&2
  fi
  exit "$status"
fi
exec pawn "${args[@]}"

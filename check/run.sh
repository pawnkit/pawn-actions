#!/usr/bin/env bash
set -euo pipefail

args=(check --project "$PAWN_PROJECT" --jobs "$PAWN_JOBS" --output "$PAWN_OUTPUT")
if [[ -n "$PAWN_ONLY" ]]; then args+=(--only "$PAWN_ONLY"); fi
if [[ -n "$PAWN_SKIP" ]]; then args+=(--skip "$PAWN_SKIP"); fi

if [[ -n "$PAWN_RESULT_FILE" ]]; then
  mkdir -p "$(dirname "$PAWN_RESULT_FILE")"
  set +e
  pawn "${args[@]}" > "$PAWN_RESULT_FILE"
  status=$?
  set -e
  exit "$status"
fi
pawn "${args[@]}"

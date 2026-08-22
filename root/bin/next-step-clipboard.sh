#!/usr/bin/env bash
set -euo pipefail

# next-step-clipboard.sh -- Next Step v1.0
#
# CONTRACT
#   Usage: <command> | next-step-clipboard.sh
#   Reads stdin, echoes it to stdout unchanged (so it's visible in the
#   terminal exactly as before), and additionally copies the same
#   content to the platform clipboard. This is the ONLY file in the
#   entire protocol with platform-specific code -- exactly one dispatch
#   line below. Every other script in the protocol core stays
#   platform-agnostic; do not add platform checks anywhere else.
#
#   Fires a PRE_CLIPBOARD hook before the platform dispatch line, using
#   the same lookup order the engine's hooks package uses:
#   NEXT_STEP_WORKSPACE_ROOT (if set) first, then
#   NEXT_STEP_HOME/hooks/. No-op if neither exists and is executable --
#   never blocks output.
#
#   Exit codes:
#     0  always -- a clipboard failure must never prevent the human from
#        seeing the output on stdout; this script does not fail loud.
#
#   Ported from the retired ABX-STEP lineage's abx-clipboard.sh
#   (env vars renamed ABX_* -> NEXT_STEP_*, home default abx-step ->
#   next-step; the multi-file-read single-write behavior this script
#   depends on is unchanged and still the caller's responsibility -- see
#   PROTOCOL-FACTS.md's Clipboard convention section for the stage_8a
#   bug this avoids).

H="${NEXT_STEP_HOME:-$HOME/next-step}"

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
cat > "$TMP"

# ---------- PRE_CLIPBOARD hook (no-op unless installed) ----------
HOOK_PATH=""
if [ -n "${NEXT_STEP_WORKSPACE_ROOT:-}" ] && [ -x "$NEXT_STEP_WORKSPACE_ROOT/hooks/PRE_CLIPBOARD" ]; then
  HOOK_PATH="$NEXT_STEP_WORKSPACE_ROOT/hooks/PRE_CLIPBOARD"
elif [ -x "$H/hooks/PRE_CLIPBOARD" ]; then
  HOOK_PATH="$H/hooks/PRE_CLIPBOARD"
fi
if [ -n "$HOOK_PATH" ]; then
  NEXT_STEP_HOOK_NAME="PRE_CLIPBOARD" "$HOOK_PATH" < "$TMP" > /dev/null 2>&1 || true
fi

# ---------- stdout passthrough, unchanged ----------
cat "$TMP"

# ---------- the one platform-specific line ----------
if command -v termux-clipboard-set >/dev/null 2>&1; then
  termux-clipboard-set < "$TMP" 2>/dev/null || true
fi

exit 0

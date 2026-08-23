#!/usr/bin/env bash
set -euo pipefail

# next-step-bootstrap.sh -- Next Step v1.0
#
# CONTRACT
#   Usage: next-step-bootstrap.sh
#
#   Convenience wrapper for onboarding's "Begin" step
#   (root/protocol/v1.0/spec/next-step-onboarding.md). Replaces the manual
#   two-line copy-paste (resolve $H, then cat state-0.md through the
#   clipboard script) with one command and clearer failure messages before
#   anything is put on the clipboard.
#
#   This script does not itself talk to an AI agent, does not read or write
#   any workspace or session state, and is not part of the onboarding state
#   chain -- it only gets state-0.md onto the clipboard so a human can paste
#   it to whichever agent they're onboarding. Everything past that point is
#   the state chain's own responsibility (state-0.md onward), not this
#   script's.
#
#   Host-only, like every file in root/bin/ -- run by a human on the
#   trusted host device, never invoked from inside a sandboxed session.
#   root/sessions/ contains no reference to this script or to anything else
#   under root/bin/; that separation is the Phase 7 enforcement check this
#   script was written alongside, not something this script enforces itself.
#
#   Exit codes:
#     0  state-0.md was found and piped to the clipboard script
#     1  no install found at $NEXT_STEP_HOME
#     2  protocol/current does not resolve to a spec/ directory
#     3  bin/next-step-clipboard.sh is missing or not executable
#     4  protocol/current/spec/state-0.md is missing

H="${NEXT_STEP_HOME:-$HOME/next-step}"

fail() {
  local code="$1" msg="$2"
  echo "next-step-bootstrap: FAILED -- $msg" >&2
  echo "  NEXT_STEP_HOME=$H" >&2
  exit "$code"
}

[ -d "$H" ] || fail 1 "no install found at $H"

[ -d "$H/protocol/current/spec" ] || fail 2 "$H/protocol/current does not resolve to a spec/ directory (check the protocol/current symlink)"

CLIP="$H/bin/next-step-clipboard.sh"
[ -x "$CLIP" ] || fail 3 "$CLIP is missing or not executable"

STATE0="$H/protocol/current/spec/state-0.md"
[ -f "$STATE0" ] || fail 4 "$STATE0 not found"

cat "$STATE0" | "$CLIP"

echo "" >&2
echo "next-step-bootstrap: state-0.md is on the clipboard (and printed above)." >&2
echo "Paste it to the agent you're onboarding to begin." >&2

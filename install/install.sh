#!/usr/bin/env bash
set -euo pipefail

# install.sh -- Next Step v1.0 public installer
#
# CONTRACT
#   Usage: install/install.sh   (run from inside a checked-out copy of this
#   repository -- a git clone, or the source archive GitHub generates for a
#   tagged release. NOT designed for curl-pipe from an arbitrary location:
#   this script mirrors the checkout's own root/ directory onto disk, per
#   docs/architecture-overview.md Sec4 (repo <-> install mirror), and needs
#   that directory to exist beside it.)
#
#   What this does, in order:
#     1. Detects platform -> one of the release matrix targets:
#          linux/amd64   -> asset "next-step-linux-amd64"
#          linux/arm64   -> asset "next-step-linux-arm64"
#          linux/arm(v7) -> asset "next-step-linux-arm-v7"   (GOARM=7)
#        This is the asset-naming convention release.yml (Phase 9) must
#        produce; ground truth lives here since release.yml doesn't exist
#        yet. If Phase 9 changes it, update this list to match.
#     2. Fetches ONLY the matching next-step binary + its "<asset>.sha256"
#        checksum sidecar from this repo's latest GitHub Release. Never
#        next-step-runner -- see docs/security-model.md Sec2 item 3: a
#        host-only install never needs to fetch or trust it.
#     3. Verifies the checksum before the binary is placed anywhere
#        permanent. Refuses to install on any verification failure.
#     4. Copies this checkout's root/ tree into $NEXT_STEP_HOME (default
#        $HOME/next-step) -- non-destructively: files present in the
#        checkout are added/updated, anything already at $NEXT_STEP_HOME
#        that isn't part of the source tree (live workspace state,
#        sessions/active, .task-seq) is left untouched. Then places the
#        verified binary at $NEXT_STEP_HOME/bin/next-step.
#
#   Never compiles from source -- see docs/security-model.md Sec3: the
#   installed binary's provenance is always a signed/checksummed release
#   artifact, never a local build.
#
#   Unauthenticated calls to api.github.com are rate-limited (60/hr per
#   IP, shared across everyone behind the same NAT). This script doesn't
#   work around that -- if it's hit, wait and retry, or set GITHUB_TOKEN
#   and re-run (not yet wired up here; a v1.0 gap, not an oversight).
#
#   Exit codes:
#     0  installed successfully
#     1  no root/ payload found beside this script (wrong run location)
#     2  unsupported OS or architecture
#     3  neither curl nor wget available
#     4  network fetch failed (unreachable, no release published yet, etc.)
#     5  latest release has no asset matching this platform
#     6  checksum verification failed, or no checksum tool available

REPO="inscope-labs/Next-Step"
H="${NEXT_STEP_HOME:-$HOME/next-step}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SRC_ROOT="$REPO_ROOT/root"

fail() {
  local msg="$1" code="${2:-1}"
  echo "install.sh: FAILED -- $msg" >&2
  exit "$code"
}

[ -d "$SRC_ROOT" ] || fail "no root/ directory found beside this script at $SRC_ROOT -- run install.sh from inside a checked-out copy of the repository, not standalone" 1

# ---------- 1. platform detection ----------
os="$(uname -s)"
case "$os" in
  Linux) ;;
  *) fail "unsupported OS '$os' -- Next Step v1.0 targets Linux (including Termux/Android) only" 2 ;;
esac

arch="$(uname -m)"
GOARM=""
case "$arch" in
  x86_64)          target="linux-amd64" ;;
  aarch64|arm64)   target="linux-arm64" ;;
  armv7l|armv8l)   target="linux-arm-v7"; GOARM=7 ;;
  *) fail "unsupported architecture '$arch' -- release matrix covers linux/amd64, linux/arm64, linux/arm(v7) only" 2 ;;
esac

asset="next-step-${target}"
echo "install.sh: detected ${os}/${arch} -> release asset ${asset}"

# ---------- 2. fetch release binary + checksum ----------
fetch() {
  local url="$1" out="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$out"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$out"
  else
    fail "neither curl nor wget found -- install one of them and re-run" 3
  fi
}

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

api_url="https://api.github.com/repos/${REPO}/releases/latest"
fetch "$api_url" "$TMPDIR/release.json" \
  || fail "could not reach $api_url -- check network access, or that a release has been published yet (see CHANGELOG.md)" 4
[ -s "$TMPDIR/release.json" ] || fail "empty response from $api_url" 4

# Minimal JSON asset-URL extraction, no jq dependency -- matches this
# repo's stdlib-only philosophy (engine/internal/registry's doc comment
# gives the same rationale for the Go side: no proxy.golang.org access
# from this environment's egress allowlist). GitHub's API response has one
# browser_download_url per line, so a line-anchored grep is sufficient;
# this is not a general JSON parser and shouldn't become one.
extract_url() {
  local name="$1"
  # "|| true": grep exits 1 on no match, and under set -o pipefail that
  # would otherwise propagate and kill the whole script here (a real bug
  # caught in testing) before the "$bin_url"/"$sum_url" emptiness checks
  # below ever get a chance to produce the actual error message. An empty
  # result from this function is an expected, handled outcome, not a
  # script-ending failure.
  grep -o "\"browser_download_url\": *\"[^\"]*${name}\"" "$TMPDIR/release.json" \
    | head -1 \
    | sed -E 's/.*"(https:[^"]+)"/\1/' || true
}

bin_url="$(extract_url "$asset")"
sum_url="$(extract_url "${asset}.sha256")"

[ -n "$bin_url" ] || fail "no release asset matching '${asset}' found on the latest release -- this platform may not be published yet" 5
[ -n "$sum_url" ] || fail "no checksum asset matching '${asset}.sha256' found -- refusing to install an unverifiable binary" 5

fetch "$bin_url" "$TMPDIR/$asset" || fail "download failed: $bin_url" 4
fetch "$sum_url" "$TMPDIR/${asset}.sha256" || fail "download failed: $sum_url" 4

# ---------- 3. verify checksum ----------
(
  cd "$TMPDIR"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c "${asset}.sha256"
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 -c "${asset}.sha256"
  else
    fail "neither sha256sum nor shasum found -- cannot verify checksum, refusing to install unverified" 6
  fi
) || fail "checksum verification failed for $asset -- refusing to install" 6

echo "install.sh: checksum OK"

# ---------- 4. place root/ payload + binary ----------
mkdir -p "$H"
cp -R "$SRC_ROOT/." "$H/"
mkdir -p "$H/bin"
install -m 0755 "$TMPDIR/$asset" "$H/bin/next-step"

echo "install.sh: root/ payload placed at $H"
echo "install.sh: installed next-step to $H/bin/next-step"
if [ -n "$GOARM" ]; then
  echo "install.sh: note -- this is the GOARM=$GOARM (armv7) build"
fi
echo ""
echo "Add $H/bin to your PATH, then run:"
echo "  NEXT_STEP_HOME=\"$H\" \"$H/bin/next-step-bootstrap.sh\""
echo "to begin onboarding."

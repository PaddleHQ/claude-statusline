#!/bin/sh
# Must never fail the session: always exits 0 and prints at most one JSON systemMessage.

REPO="PaddleHQ/claude-statusline"
SIGNER_WORKFLOW="$REPO/.github/workflows/release-binaries.yml"
CLAUDE_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
if [ -n "${CLAUDE_PLUGIN_DATA:-}" ]; then APP_DIR="$CLAUDE_PLUGIN_DATA"; else APP_DIR="$CLAUDE_DIR/claude-statusline"; fi
BIN="$APP_DIR/bin/claude-statusline"
CHECK_FILE="$APP_DIR/last-check"
NOTICE_FILE="$APP_DIR/.no-attestation-notice"
BASE="${CLAUDE_STATUSLINE_DOWNLOAD_BASE:-https://github.com/$REPO/releases/download}"
CHECK_INTERVAL=86400
TMP=""

fail() {
  [ -z "$TMP" ] || rm -rf "$TMP"
  printf '{"systemMessage":"claude-statusline: %s"}\n' "$1"
  exit 0
}

# With a binary already installed a failed check is retried next session, so stay quiet.
soft_fail() {
  [ ! -x "$BIN" ] && fail "$1"
  [ -z "$TMP" ] || rm -rf "$TMP"
  "$BIN" setup --event=current 2>/dev/null
  exit 0
}

reported_version() {
  "$1" --version 2>/dev/null | awk 'NR == 1 && $1 == "claude-statusline" { print $2 }'
}

sha256_of() {
  if command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1"; else sha256sum "$1"; fi | awk '{ print $1 }'
}

# Succeeds when $1 is a newer x.y.z than $2; an unparseable $2 counts as older.
is_newer() {
  awk -v a="$1" -v b="$2" 'BEGIN {
    if (b !~ /^[0-9]+\.[0-9]+\.[0-9]+$/) exit 0
    split(a, x, "."); split(b, y, ".")
    for (i = 1; i <= 3; i++) {
      if (x[i] + 0 > y[i] + 0) exit 0
      if (x[i] + 0 < y[i] + 0) exit 1
    }
    exit 1
  }'
}

record_check() {
  date +%s > "$CHECK_FILE" 2>/dev/null
}

if [ -x "$BIN" ]; then
  last=$(cat "$CHECK_FILE" 2>/dev/null)
  case "$last" in '' | *[!0-9]*) last=0 ;; esac
  now=$(date +%s)
  if [ "$last" -le "$now" ] && [ $((now - last)) -lt "$CHECK_INTERVAL" ]; then
    "$BIN" setup --event=current 2>/dev/null
    exit 0
  fi
fi

case "$(uname -s)" in Darwin) OS=darwin ;; Linux) OS=linux ;; *) OS="" ;; esac
case "$(uname -m)" in arm64 | aarch64) ARCH=arm64 ;; x86_64 | amd64) ARCH=amd64 ;; *) ARCH="" ;; esac
if [ -z "$OS" ] || [ -z "$ARCH" ]; then
  plat=$(printf '%s/%s' "$(uname -s)" "$(uname -m)" | tr -cd 'A-Za-z0-9_./-')
  soft_fail "unsupported platform $plat; only macOS and Linux on arm64 or amd64 are supported."
fi
command -v curl >/dev/null 2>&1 || soft_fail "curl was not found, so the latest release could not be checked."
command -v shasum >/dev/null 2>&1 || command -v sha256sum >/dev/null 2>&1 || soft_fail "neither shasum nor sha256sum was found, so downloads cannot be verified and nothing was installed."

if [ -n "${CLAUDE_STATUSLINE_LATEST_URL:-}" ]; then
  RAW=$(curl -fsSL --proto '=file' --max-time 10 "$CLAUDE_STATUSLINE_LATEST_URL" 2>/dev/null)
else
  RAW=$(curl -fsSL --proto '=https' --proto-redir '=https' --max-time 10 -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest" 2>/dev/null)
fi
TAG=$(printf '%s' "$RAW" | tr -d '[:space:]')
TAG="${TAG##*/}"
[ -n "$TAG" ] || soft_fail "could not find the latest release; will retry next session."
printf '%s' "$TAG" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$' || soft_fail "the latest release tag is not a valid version, so nothing was installed."
V="${TAG#v}"

INSTALLED=""
[ ! -x "$BIN" ] || INSTALLED=$(reported_version "$BIN")
if [ -x "$BIN" ] && ! is_newer "$V" "$INSTALLED"; then
  mkdir -p "$APP_DIR" && record_check
  "$BIN" setup --event=current 2>/dev/null
  exit 0
fi

EVENT=installed
[ ! -x "$BIN" ] || EVENT=updated
ARCHIVE="claude-statusline_${V}_${OS}_${ARCH}.tar.gz"

mkdir -p "$APP_DIR/bin" || fail "could not create the install directory."
find "$APP_DIR" -maxdepth 1 -name 'tmp.*' -mmin +10 -exec rm -rf {} + 2>/dev/null
TMP=$(mktemp -d "$APP_DIR/tmp.XXXXXX") || fail "could not create a temporary directory in the install directory."
trap 'rm -rf "$TMP"' EXIT
curl -fsSL --proto '=https,file' --proto-redir '=https' --max-time 20 -o "$TMP/$ARCHIVE" "$BASE/$TAG/$ARCHIVE" 2>/dev/null || soft_fail "downloading $TAG failed; will retry next session."
curl -fsSL --proto '=https,file' --proto-redir '=https' --max-time 10 -o "$TMP/checksums.txt" "$BASE/$TAG/checksums.txt" 2>/dev/null || soft_fail "downloading the checksums for $TAG failed; will retry next session."

WANT=$(awk -v f="$ARCHIVE" '$2 == f { print $1; exit }' "$TMP/checksums.txt" 2>/dev/null)
printf '%s' "$WANT" | grep -Eq '^[0-9a-f]{64}$' || fail "the $TAG release has no checksum for $ARCHIVE, so nothing was installed."
GOT=$(sha256_of "$TMP/$ARCHIVE")
[ "$GOT" = "$WANT" ] || fail "the $TAG download FAILED checksum verification and was NOT installed."

PROVENANCE=0
# A gh older than the attestation command counts as no gh, not as a failed verification.
if command -v gh >/dev/null 2>&1 && gh attestation --help >/dev/null 2>&1; then
  # With the release's bundle gh calls no API, so a token GitHub rejects cannot block the install.
  curl -fsSL --proto '=https,file' --proto-redir '=https' --max-time 10 -o "$TMP/provenance.jsonl" "$BASE/$TAG/provenance.jsonl" 2>/dev/null || soft_fail "downloading the provenance bundle for $TAG failed; will retry next session."
  GH_HOST=github.com gh attestation verify "$TMP/$ARCHIVE" --bundle "$TMP/provenance.jsonl" --signer-workflow "$SIGNER_WORKFLOW" --repo "$REPO" >/dev/null 2>&1 || fail "provenance verification of $TAG FAILED and it was NOT installed."
  PROVENANCE=1
fi

mkdir "$TMP/x" && tar -xzf "$TMP/$ARCHIVE" -C "$TMP/x" claude-statusline 2>/dev/null && [ -f "$TMP/x/claude-statusline" ] && [ ! -L "$TMP/x/claude-statusline" ] || fail "could not extract the binary from the $TAG archive; nothing was installed."
chmod 755 "$TMP/x/claude-statusline"
[ "$(reported_version "$TMP/x/claude-statusline")" = "$V" ] || fail "the downloaded binary did not report version $V; nothing was installed."
mv -f "$TMP/x/claude-statusline" "$BIN" || fail "could not place the binary in the install directory."

rm -rf "$TMP"
record_check
OUT=$("$BIN" setup --event="$EVENT" 2>/dev/null)
if [ "$PROVENANCE" = 0 ] && [ ! -e "$NOTICE_FILE" ]; then
  NOTICE="Installed $TAG after a checksum check only. Install the GitHub CLI (gh) to also verify build provenance on future updates."
  : > "$NOTICE_FILE"
  case "$OUT" in
    '') printf '{"systemMessage":"claude-statusline: %s"}\n' "$NOTICE" ;;
    *'"}') printf '%s\n' "$OUT" | sed "s/\"}\$/ $NOTICE\"}/" ;;
    *) printf '%s\n' "$OUT" ;;
  esac
elif [ -n "$OUT" ]; then
  printf '%s\n' "$OUT"
fi
exit 0

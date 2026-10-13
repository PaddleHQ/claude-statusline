#!/bin/sh
# Must never fail the session: always exits 0 and prints at most one JSON systemMessage.

CLAUDE_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
APP_DIR="$CLAUDE_DIR/claude-statusline"
BIN="$APP_DIR/bin/claude-statusline"
PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." 2>/dev/null && pwd)}"
BASE="${CLAUDE_STATUSLINE_DOWNLOAD_BASE:-https://github.com/PaddleHQ/claude-statusline/releases/download}"
TMP=""

fail() {
  [ -z "$TMP" ] || rm -rf "$TMP"
  printf '{"systemMessage":"claude-statusline: %s"}\n' "$1"
  exit 0
}

reported_version() {
  "$1" --version 2>/dev/null | awk 'NR == 1 && $1 == "claude-statusline" { print $2 }'
}

sha256_of() {
  if command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1"; else sha256sum "$1"; fi | awk '{ print $1 }'
}

V=$(tr -d '[:space:]' < "$PLUGIN_ROOT/release/version" 2>/dev/null)
[ -n "$V" ] || exit 0
printf '%s' "$V" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$' || fail "invalid pinned version in release/version, so nothing was installed."

case "$(uname -s)" in Darwin) OS=darwin ;; Linux) OS=linux ;; *) OS="" ;; esac
case "$(uname -m)" in arm64 | aarch64) ARCH=arm64 ;; x86_64 | amd64) ARCH=amd64 ;; *) ARCH="" ;; esac
if [ -z "$OS" ] || [ -z "$ARCH" ]; then
  plat=$(printf '%s/%s' "$(uname -s)" "$(uname -m)" | tr -cd 'A-Za-z0-9_./-')
  fail "unsupported platform $plat; only macOS and Linux on arm64 or amd64 are supported."
fi

if [ -x "$BIN" ] && [ "$(reported_version "$BIN")" = "$V" ]; then
  "$BIN" setup --event=current 2>/dev/null
  exit 0
fi

EVENT=installed
[ ! -x "$BIN" ] || EVENT=updated
ARCHIVE="claude-statusline_${V}_${OS}_${ARCH}.tar.gz"
WANT=$(awk -v f="$ARCHIVE" '$2 == f { print $1; exit }' "$PLUGIN_ROOT/release/checksums.txt" 2>/dev/null)
printf '%s' "$WANT" | grep -Eq '^[0-9a-f]{64}$' || fail "no checksum is pinned for $ARCHIVE, so nothing was installed."
command -v curl >/dev/null 2>&1 || fail "curl was not found, so v$V could not be downloaded."
command -v shasum >/dev/null 2>&1 || command -v sha256sum >/dev/null 2>&1 || fail "neither shasum nor sha256sum was found, so the download cannot be verified and was not installed."

mkdir -p "$APP_DIR/bin" || fail "could not create the install directory."
TMP=$(mktemp -d "$APP_DIR/tmp.XXXXXX") || fail "could not create a temporary directory under the Claude config directory."
trap 'rm -rf "$TMP"' EXIT
curl -fsSL --proto '=https,file' --proto-redir '=https' --max-time 20 -o "$TMP/$ARCHIVE" "$BASE/v$V/$ARCHIVE" 2>/dev/null || fail "downloading v$V failed; will retry next session."
GOT=$(sha256_of "$TMP/$ARCHIVE")
[ "$GOT" = "$WANT" ] || fail "the v$V download FAILED checksum verification and was NOT installed."
mkdir "$TMP/x" && tar -xzf "$TMP/$ARCHIVE" -C "$TMP/x" claude-statusline 2>/dev/null && [ -f "$TMP/x/claude-statusline" ] && [ ! -L "$TMP/x/claude-statusline" ] || fail "could not extract the binary from the v$V archive; nothing was installed."
chmod 755 "$TMP/x/claude-statusline"
[ "$(reported_version "$TMP/x/claude-statusline")" = "$V" ] || fail "the downloaded binary did not report version $V; nothing was installed."
mv -f "$TMP/x/claude-statusline" "$BIN" || fail "could not place the binary in the Claude config directory."

rm -rf "$TMP"
"$BIN" setup --event="$EVENT" 2>/dev/null
exit 0

#!/bin/sh
# Must never fail the session: always exits 0 and prints at most one JSON systemMessage.

CLAUDE_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
APP_DIR="$CLAUDE_DIR/claude-statusline"
BIN="$APP_DIR/bin/claude-statusline"
SETTINGS="$CLAUDE_DIR/settings.json"
SENTINEL="$APP_DIR/.bootstrapped"
PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." 2>/dev/null && pwd)}"
DOWNLOAD_BASE="${CLAUDE_STATUSLINE_DOWNLOAD_BASE:-https://github.com/PaddleHQ/claude-statusline/releases/download}"

MSG=""
TMP=""

add_msg() {
  if [ -n "$MSG" ]; then
    MSG="$MSG $1"
  else
    MSG="$1"
  fi
}

emit() {
  [ -n "$MSG" ] || return 0
  if command -v jq >/dev/null 2>&1; then
    jq -nc --arg m "$MSG" '{systemMessage: $m}'
  else
    escaped=$(printf '%s' "$MSG" | sed 's/\\/\\\\/g; s/"/\\"/g')
    printf '{"systemMessage":"%s"}\n' "$escaped"
  fi
}

reported_version() {
  "$1" --version 2>/dev/null | awk 'NR == 1 && $1 == "claude-statusline" { print $2 }'
}

sha256_of() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{ print $1 }'
  else
    sha256sum "$1" | awk '{ print $1 }'
  fi
}

install_binary() {
  archive="claude-statusline_${V}_${OS}_${ARCH}.tar.gz"

  want=$(awk -v f="$archive" '$2 == f { print $1; exit }' "$PLUGIN_ROOT/release/checksums.txt" 2>/dev/null)
  if ! printf '%s' "$want" | grep -Eq '^[0-9a-f]{64}$'; then
    add_msg "claude-statusline: no checksum is pinned for $archive, so nothing was installed."
    return 1
  fi
  if ! command -v curl >/dev/null 2>&1; then
    add_msg "claude-statusline: curl was not found, so v$V could not be downloaded."
    return 1
  fi
  if ! command -v shasum >/dev/null 2>&1 && ! command -v sha256sum >/dev/null 2>&1; then
    add_msg "claude-statusline: neither shasum nor sha256sum was found, so the download cannot be verified and was not installed."
    return 1
  fi

  mkdir -p "$APP_DIR/bin" || return 1
  TMP=$(mktemp -d "$APP_DIR/tmp.XXXXXX") || return 1
  trap 'rm -rf "$TMP"' EXIT

  if ! curl -fsSL --proto '=https,file' --proto-redir '=https' --max-time 20 \
    -o "$TMP/$archive" "$DOWNLOAD_BASE/v$V/$archive" 2>/dev/null; then
    add_msg "claude-statusline: downloading v$V failed; will retry next session."
    return 1
  fi

  got=$(sha256_of "$TMP/$archive")
  if [ "$got" != "$want" ]; then
    add_msg "claude-statusline: the v$V download FAILED checksum verification and was NOT installed."
    return 1
  fi

  mkdir "$TMP/x" || return 1
  if ! tar -xzf "$TMP/$archive" -C "$TMP/x" claude-statusline 2>/dev/null \
    || [ ! -f "$TMP/x/claude-statusline" ] || [ -L "$TMP/x/claude-statusline" ]; then
    add_msg "claude-statusline: could not extract the binary from the v$V archive; nothing was installed."
    return 1
  fi
  chmod 755 "$TMP/x/claude-statusline" || return 1
  if [ "$(reported_version "$TMP/x/claude-statusline")" != "$V" ]; then
    add_msg "claude-statusline: the downloaded binary did not report version $V; nothing was installed."
    return 1
  fi
  if ! mv -f "$TMP/x/claude-statusline" "$BIN"; then
    add_msg "claude-statusline: could not place the binary at $BIN."
    return 1
  fi
  return 0
}

wire_settings() {
  [ ! -e "$SENTINEL" ] || return 0

  if ! command -v jq >/dev/null 2>&1; then
    add_msg "claude-statusline v$V installed. jq was not found, so settings.json was not edited. If you have no statusLine, add {\"statusLine\":{\"type\":\"command\",\"command\":\"$BIN\",\"refreshInterval\":60}} to $SETTINGS."
    touch "$SENTINEL"
    return 0
  fi

  if [ ! -f "$SETTINGS" ]; then
    mkdir -p "$CLAUDE_DIR" && printf '{}\n' > "$SETTINGS" || return 0
  fi
  if ! jq -e 'type == "object"' "$SETTINGS" >/dev/null 2>&1; then
    [ "$INSTALLED" = 1 ] && add_msg "claude-statusline v$V installed, but $SETTINGS is not valid JSON so it was left alone. It will retry next session."
    return 0
  fi

  if jq -e 'has("statusLine")' "$SETTINGS" >/dev/null 2>&1; then
    if [ "$(jq -r '.statusLine.command // ""' "$SETTINGS")" = "$BIN" ]; then
      [ "$INSTALLED" = 1 ] && add_msg "claude-statusline v$V installed."
    else
      add_msg "claude-statusline v$V installed, but you already have a statusLine so it was left unchanged. To switch, set statusLine.command in $SETTINGS to $BIN."
    fi
    touch "$SENTINEL"
    return 0
  fi

  cp "$SETTINGS" "$SETTINGS.bak" || return 0
  tmp=$(mktemp "$CLAUDE_DIR/settings.XXXXXX") || return 0
  if jq --arg cmd "$BIN" '.statusLine = {type: "command", command: $cmd, refreshInterval: 60}' "$SETTINGS" > "$tmp" \
    && mv -f "$tmp" "$SETTINGS"; then
    touch "$SENTINEL"
    add_msg "claude-statusline v$V installed and enabled. It appears after your first message. Your previous settings are backed up at $SETTINGS.bak. To disable it, remove the statusLine block from settings.json."
  else
    rm -f "$tmp"
  fi
}

main() {
  V=$(tr -d '[:space:]' < "$PLUGIN_ROOT/release/version" 2>/dev/null)
  [ -n "$V" ] || return 0
  if ! printf '%s' "$V" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    add_msg "claude-statusline: invalid pinned version in release/version, so nothing was installed."
    return 0
  fi

  case "$(uname -s)" in
    Darwin) OS=darwin ;;
    Linux) OS=linux ;;
    *) OS="" ;;
  esac
  case "$(uname -m)" in
    arm64 | aarch64) ARCH=arm64 ;;
    x86_64 | amd64) ARCH=amd64 ;;
    *) ARCH="" ;;
  esac
  if [ -z "$OS" ] || [ -z "$ARCH" ]; then
    add_msg "claude-statusline: unsupported platform $(uname -s)/$(uname -m); only macOS and Linux on arm64 or amd64 are supported."
    return 0
  fi

  INSTALLED=0
  if [ -x "$BIN" ] && [ "$(reported_version "$BIN")" = "$V" ]; then
    :
  else
    had_bin=0
    [ -x "$BIN" ] && had_bin=1
    install_binary || return 0
    INSTALLED=1
    if [ "$had_bin" = 1 ] && [ -e "$SENTINEL" ]; then
      add_msg "claude-statusline updated to v$V."
    fi
  fi

  wire_settings
}

main
emit
exit 0

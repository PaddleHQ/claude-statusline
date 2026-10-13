#!/bin/sh
# claude-statusline: segment-based status line for Claude Code.
# Reads the session JSON on stdin and prints a single line. Always exits 0.

set -f

SCRIPT_DIR=$(cd -- "$(dirname -- "$0")" && pwd -P)
CLAUDE_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
APP_DIR="$CLAUDE_DIR/claude-statusline"
CONFIG_FILE="$APP_DIR/config"
CACHE_DIR="$APP_DIR/cache"
CACHE_FILE="$CACHE_DIR/ratelimits"
DEFAULTS_FILE="$SCRIPT_DIR/defaults.conf"

US=$(printf '\037')

# shellcheck source=lib/util.sh
. "$SCRIPT_DIR/lib/util.sh"
# shellcheck source=lib/config.sh
. "$SCRIPT_DIR/lib/config.sh"
# shellcheck source=lib/theme.sh
. "$SCRIPT_DIR/lib/theme.sh"
# shellcheck source=lib/rate.sh
. "$SCRIPT_DIR/lib/rate.sh"
# shellcheck source=lib/bar.sh
. "$SCRIPT_DIR/lib/bar.sh"
# shellcheck source=lib/payload.sh
. "$SCRIPT_DIR/lib/payload.sh"
# shellcheck source=lib/render.sh
. "$SCRIPT_DIR/lib/render.sh"

main() {
  input=$(cat)
  command -v jq >/dev/null 2>&1 || return 0
  [ -r "$DEFAULTS_FILE" ] || return 0
  load_config_file "$DEFAULTS_FILE" DEF__
  load_config_file "$CONFIG_FILE" CFG__
  apply_theme
  set_now
  read_payload "$input"
  merge_rate_limits
  render_line
}

main
exit 0

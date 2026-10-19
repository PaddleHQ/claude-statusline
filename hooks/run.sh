#!/bin/sh
# Runs the installed claude-statusline with the given arguments, for the plugin's slash commands.

CLAUDE_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"

for BIN in \
  "${CLAUDE_PLUGIN_DATA:-$CLAUDE_DIR/plugins/data/statusline-claude-statusline}/bin/claude-statusline" \
  "$CLAUDE_DIR/claude-statusline/bin/claude-statusline"; do
  [ -x "$BIN" ] && exec "$BIN" "$@"
done
echo "NOT-INSTALLED"

#!/bin/sh
# Runs the installed claude-statusline with the given arguments, for the plugin's slash commands.
# A non-zero exit status is printed on the last line, because a command's output is all Claude sees.

CLAUDE_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"

for BIN in \
  "${CLAUDE_PLUGIN_DATA:-$CLAUDE_DIR/plugins/data/statusline-claude-statusline}/bin/claude-statusline" \
  "$CLAUDE_DIR/claude-statusline/bin/claude-statusline"; do
  if [ -x "$BIN" ]; then
    "$BIN" "$@"
    status=$?
    [ "$status" -eq 0 ] || echo "exit status: $status"
    exit "$status"
  fi
done
echo "NOT-INSTALLED"

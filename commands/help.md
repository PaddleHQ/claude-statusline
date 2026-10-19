---
description: Explain what your status line shows, from your own settings
allowed-tools: Bash(*/claude-statusline/bin/claude-statusline help)
disable-model-invocation: true
---

Output of the status line's help command:

!`"${CLAUDE_CONFIG_DIR:-$HOME/.claude}/claude-statusline/bin/claude-statusline" help 2>&1 || echo "NOT-INSTALLED"`

Present that output to the user as a short list, grouped as it is: shown now, custom segments, also available. Keep every entry and its description, and the config file and config error lines at the end, and add nothing that is not in it. If it says NOT-INSTALLED, say the plugin has not installed the status line binary yet and that they should start a new session, then run this command again.

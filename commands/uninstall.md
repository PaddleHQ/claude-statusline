---
description: Remove the status line from your settings before uninstalling the plugin
allowed-tools: Bash(sh *hooks/run.sh uninstall)
disable-model-invocation: true
---

Output of the status line's uninstall command:

!`sh "${CLAUDE_PLUGIN_ROOT}/hooks/run.sh" uninstall 2>&1`

Tell the user what the output says was removed or left alone, in a few short lines, and that the lines will not be added back while the plugin is installed (`claude-statusline setup` turns them on again). Then tell them to run `/plugin uninstall statusline@claude-statusline` themselves to finish: that removes the plugin and its data directory, and you must not run it for them. Their `config.yaml`, `statusline.d` hooks and `usage-latest.json` are never deleted. If the output says NOT-INSTALLED, the status line binary is not installed, so there is nothing for this command to remove: tell them to run `/plugin uninstall statusline@claude-statusline`, and to delete any `statusLine` or `subagentStatusLine` entry of theirs that still points at claude-statusline from `~/.claude/settings.json`.

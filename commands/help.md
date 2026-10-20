---
description: Explain what your status line shows, from your own settings
allowed-tools: Bash(sh "*/hooks/run.sh" help)
disable-model-invocation: true
---

Output of the status line's help command:

!`sh "${CLAUDE_PLUGIN_ROOT}/hooks/run.sh" help 2>&1`

Present that output to the user as a short list, grouped as it is: shown now, custom segments, also available. Keep every entry and its description, and add nothing that is not in it. If it says NOT-INSTALLED, say the plugin has not installed the status line binary yet and that they should start a new session, then run this command again.

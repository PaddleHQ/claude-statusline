package statusline_test

import (
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/statusline"

	"github.com/stretchr/testify/assert"
)

// TestHelp pins what the help command lists: the segments shown, the custom segments and how each is described,
// the built-ins left out, and the settings the descriptions reflect.
func TestHelp(t *testing.T) {
	t.Parallel()

	tests := []scenario{
		{
			name:   `A statusline-help directive after the first 20 lines is ignored`,
			config: `segments: [path]`,
			executables: map[string]string{
				`statusline.d/late.sh`: `#!/bin/sh
#
#
#
#
#
#
#
#
#
#
#
#
#
#
#
#
#
#
#
#
#
# statusline-help: too late
printf nothing`,
			},
			want: `Your status line, as currently configured.

Shown now, in order:
  path: The working directory, with your home directory shown as ~, followed by the git branch in brackets.

Custom segments:
  late (late.sh, slot ctx): custom segment from late.sh

Also available, not in your segments list:
  model: The model in use and its effort level, as in Opus 4.7 [high]. It reads Haiku (skill), for example, while a skill or subagent runs a different model than the session's.
  ctx: How full the context window is. Amber from 50%, red from 90%.
  tokens: Session token totals: input after the down arrow, output after the up arrow.
  cache: How much of the latest request's input was read from the prompt cache. Red below 5%.
  rate5h: Your 5 hour usage, with a marker at an even pace. Amber when more than 20 points off it, red beyond 40 points or from 95% used.
  rate7d: Your 7 day usage, with a marker at an even pace. Amber when more than 10 points off it, red beyond 20 points or from 95% used.
  boot: The prompt size on the first turn, before you said anything.
  skills: The size of the always-loaded skill listing, in tokens.
  skills_stripped: Red when skills lost their descriptions to fit the budget; shows how many.
  agent_skills: Amber when a subagent lost skill descriptions; shows how many.
  mem: Red when a SessionStart hook output was too large to inject.

Config file: @TMP@/claude-statusline/config.yaml
Documentation: https://github.com/PaddleHQ/claude-statusline/blob/main/docs/customising.md`,
		},
		{
			name:   `A hook with a statusline-help directive is described by it, and one without says which file it came from`,
			config: `segments: [path]`,
			executables: map[string]string{
				`statusline.d/10-backup.sh`: `#!/bin/sh
# statusline-help: Shows when the backup is stale
# statusline-slot: end
printf nothing`,
				`statusline.d/20-plain.sh`: `#!/bin/sh
printf nothing`,
			},
			want: `Your status line, as currently configured.

Shown now, in order:
  path: The working directory, with your home directory shown as ~, followed by the git branch in brackets.

Custom segments:
  backup (10-backup.sh, slot end): Shows when the backup is stale
  plain (20-plain.sh, slot ctx): custom segment from 20-plain.sh

Also available, not in your segments list:
  model: The model in use and its effort level, as in Opus 4.7 [high]. It reads Haiku (skill), for example, while a skill or subagent runs a different model than the session's.
  ctx: How full the context window is. Amber from 50%, red from 90%.
  tokens: Session token totals: input after the down arrow, output after the up arrow.
  cache: How much of the latest request's input was read from the prompt cache. Red below 5%.
  rate5h: Your 5 hour usage, with a marker at an even pace. Amber when more than 20 points off it, red beyond 40 points or from 95% used.
  rate7d: Your 7 day usage, with a marker at an even pace. Amber when more than 10 points off it, red beyond 20 points or from 95% used.
  boot: The prompt size on the first turn, before you said anything.
  skills: The size of the always-loaded skill listing, in tokens.
  skills_stripped: Red when skills lost their descriptions to fit the budget; shows how many.
  agent_skills: Amber when a subagent lost skill descriptions; shows how many.
  mem: Red when a SessionStart hook output was too large to inject.

Config file: @TMP@/claude-statusline/config.yaml
Documentation: https://github.com/PaddleHQ/claude-statusline/blob/main/docs/customising.md`,
		},
		{
			name:   `A segments list shows only those segments as shown now, in the listed order, and offers the rest as available`,
			config: `segments: [path, model, rate5h]`,
			want: `Your status line, as currently configured.

Shown now, in order:
  path: The working directory, with your home directory shown as ~, followed by the git branch in brackets.
  model: The model in use and its effort level, as in Opus 4.7 [high]. It reads Haiku (skill), for example, while a skill or subagent runs a different model than the session's.
  rate5h: Your 5 hour usage, with a marker at an even pace. Amber when more than 20 points off it, red beyond 40 points or from 95% used.

Also available, not in your segments list:
  ctx: How full the context window is. Amber from 50%, red from 90%.
  tokens: Session token totals: input after the down arrow, output after the up arrow.
  cache: How much of the latest request's input was read from the prompt cache. Red below 5%.
  rate7d: Your 7 day usage, with a marker at an even pace. Amber when more than 10 points off it, red beyond 20 points or from 95% used.
  boot: The prompt size on the first turn, before you said anything.
  skills: The size of the always-loaded skill listing, in tokens.
  skills_stripped: Red when skills lost their descriptions to fit the budget; shows how many.
  agent_skills: Amber when a subagent lost skill descriptions; shows how many.
  mem: Red when a SessionStart hook output was too large to inject.

Config file: @TMP@/claude-statusline/config.yaml
Documentation: https://github.com/PaddleHQ/claude-statusline/blob/main/docs/customising.md`,
		},
		{
			name: `A changed threshold is reflected in the help text`,
			config: `segments: [ctx, cache]
ctx:
  warn: 55
  bad: 80
cache:
  warn: 20`,
			want: `Your status line, as currently configured.

Shown now, in order:
  ctx: How full the context window is. Amber from 55%, red from 80%.
  cache: How much of the latest request's input was read from the prompt cache. Red below 20%.

Also available, not in your segments list:
  path: The working directory, with your home directory shown as ~, followed by the git branch in brackets.
  model: The model in use and its effort level, as in Opus 4.7 [high]. It reads Haiku (skill), for example, while a skill or subagent runs a different model than the session's.
  tokens: Session token totals: input after the down arrow, output after the up arrow.
  rate5h: Your 5 hour usage, with a marker at an even pace. Amber when more than 20 points off it, red beyond 40 points or from 95% used.
  rate7d: Your 7 day usage, with a marker at an even pace. Amber when more than 10 points off it, red beyond 20 points or from 95% used.
  boot: The prompt size on the first turn, before you said anything.
  skills: The size of the always-loaded skill listing, in tokens.
  skills_stripped: Red when skills lost their descriptions to fit the budget; shows how many.
  agent_skills: Amber when a subagent lost skill descriptions; shows how many.
  mem: Red when a SessionStart hook output was too large to inject.

Config file: @TMP@/claude-statusline/config.yaml
Documentation: https://github.com/PaddleHQ/claude-statusline/blob/main/docs/customising.md`,
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, getenv := tt.home(t)
			got := statusline.Help(getenv)
			assert.Equal(t, tt.expected(dir), strings.TrimSuffix(got, "\n"))
			assert.NotContains(t, got, "\033", "help is plain text")
		})
	}
}

package statusline_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/statusline"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
  model: The model in use and its effort level, as in Opus 4.7 [high]. It reads Haiku 4.5 (skill), for example, while a skill or subagent runs a different model than the session's.
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
  model: The model in use and its effort level, as in Opus 4.7 [high]. It reads Haiku 4.5 (skill), for example, while a skill or subagent runs a different model than the session's.
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
  model: The model in use and its effort level, as in Opus 4.7 [high]. It reads Haiku 4.5 (skill), for example, while a skill or subagent runs a different model than the session's.
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
  model: The model in use and its effort level, as in Opus 4.7 [high]. It reads Haiku 4.5 (skill), for example, while a skill or subagent runs a different model than the session's.
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

func TestHelpStripsTerminalEscapes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	hooks := filepath.Join(dir, "hooks")
	require.NoError(t, os.Mkdir(hooks, 0o750))
	name := "b\x1b[31mRED.sh"
	require.NoError(t, os.WriteFile(filepath.Join(hooks, name), []byte("#!/bin/sh\n"), 0o700)) //nolint:gosec // test hook must be executable
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "claude-statusline"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "claude-statusline", "config.yaml"), []byte("segments: [path]\npath:\n  strip: [\"\\e]0;pwn\\a/x\"]\n"), 0o600))

	got := statusline.Help(func(k string) string {
		return map[string]string{"STATUSLINE_HOOK_DIR": hooks, "CLAUDE_CONFIG_DIR": dir}[k]
	})

	assert.NotContains(t, got, "\x1b")
	assert.NotContains(t, got, "\x07")
	assert.Contains(t, got, "RED.sh")
}

func TestHelpRemovesInvisibleFormatCharacters(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	hooks := filepath.Join(dir, "hooks")
	require.NoError(t, os.Mkdir(hooks, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(hooks, "ab"+string(rune(0x202e))+"cd.sh"), []byte("#!/bin/sh\n"), 0o700)) //nolint:gosec // test hook must be executable

	got := statusline.Help(func(k string) string {
		return map[string]string{"STATUSLINE_HOOK_DIR": hooks, "CLAUDE_CONFIG_DIR": dir}[k]
	})

	assert.NotContains(t, got, "\u202e")
	assert.Contains(t, got, "abcd.sh")
}

func TestHelpShowsConfigProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T, path string)
		want    []string
		notWant []string
	}{
		{
			name:    "a broken config names the line and the reason under the file",
			setup:   writeText("hooks:\n  ttl: 60\n timeout: 2\n"),
			want:    []string{"Config file: @TMP@/claude-statusline/config.yaml\nConfig error: @TMP@/claude-statusline/config.yaml:2 did not find expected key\n"},
			notWant: []string{"not created yet"},
		},
		{
			name: "a directory at the config path is an error, not a file yet to be created",
			setup: func(t *testing.T, path string) {
				t.Helper()
				require.NoError(t, os.Mkdir(path, 0o700))
			},
			want:    []string{"Config file: @TMP@/claude-statusline/config.yaml\nConfig error: @TMP@/claude-statusline/config.yaml: not a regular file\n"},
			notWant: []string{"not created yet"},
		},
		{
			name:    "a valid config has no error line",
			setup:   writeText("segments: [model]\n"),
			want:    []string{"Config file: @TMP@/claude-statusline/config.yaml\n"},
			notWant: []string{"Config error", "not created yet"},
		},
		{
			name:    "a missing config has no error line",
			want:    []string{"Config file: @TMP@/claude-statusline/config.yaml (not created yet, so the defaults apply)\n"},
			notWant: []string{"Config error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			home := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(home, "claude-statusline"), 0o700))
			if tt.setup != nil {
				tt.setup(t, filepath.Join(home, "claude-statusline", "config.yaml"))
			}

			got := statusline.Help(func(k string) string {
				return map[string]string{"HOME": home, "CLAUDE_CONFIG_DIR": home, "STATUSLINE_HOOK_DIR": filepath.Join(home, "none")}[k]
			})
			got = strings.ReplaceAll(got, home, "@TMP@")

			for _, w := range tt.want {
				assert.Contains(t, got, w)
			}
			for _, n := range tt.notWant {
				assert.NotContains(t, got, n)
			}
		})
	}
}

func TestHelpCleansConfigError(t *testing.T) {
	t.Parallel()

	home := t.TempDir() + "/a\x1b[31m\u202eb"
	require.NoError(t, os.MkdirAll(filepath.Join(home, "claude-statusline"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(home, "claude-statusline", "config.yaml"), []byte("hooks:\n  ttl: 60\n timeout: 2\n"), 0o600))

	got := statusline.Help(func(k string) string {
		return map[string]string{"HOME": home, "CLAUDE_CONFIG_DIR": home, "STATUSLINE_HOOK_DIR": filepath.Join(home, "none")}[k]
	})

	assert.NotContains(t, got, "\x1b")
	assert.NotContains(t, got, "\u202e")
	assert.Contains(t, got, "a[31mb/claude-statusline/config.yaml:2 did not find expected key\n")
}

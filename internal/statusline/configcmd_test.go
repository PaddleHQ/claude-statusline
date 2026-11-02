package statusline_test

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/statusline"
	"github.com/PaddleHQ/claude-statusline/segments"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	got := statusline.DefaultConfig()

	assert.Equal(t, config.CommentedDefaults(segments.Defaults()...), got)

	last := -1
	for _, section := range segments.Defaults() {
		first, _, _ := strings.Cut(strings.TrimSpace(section), "\n")
		for line := range strings.SplitSeq(section, "\n") {
			if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, " ") {
				first = line
				break
			}
		}
		at := strings.Index(got, "\n# "+first)
		require.GreaterOrEqual(t, at, 0, "section starting %q is missing", first)
		assert.Greater(t, at, last, "section starting %q is out of order", first)
		last = at
	}
}

func TestConfigCheck(t *testing.T) {
	t.Parallel()

	const file = "Config file: ~/claude-statusline/config.yaml\n"

	tests := []struct {
		name    string
		setup   func(t *testing.T, path string)
		noDir   bool
		want    string
		wantBad bool
	}{
		{
			name:  "a valid file parses without errors",
			setup: writeText("segments: [model]\nhooks:\n  ttl: 30\n"),
			want:  file + "Parsed without errors.\n",
		},
		{
			name:  "a valid file lists the settings no default has, and still passes",
			setup: writeText("# hooks:\nttl: 60\nmodel:\n  shade: red\nsegments: [model]\n"),
			want:  file + "Parsed without errors.\n\nUnknown settings (ignored):\n  model.shade\n  ttl\n",
		},
		{
			name: "a missing file means the defaults",
			want: file + "Does not exist, so the defaults apply.\n",
		},
		{
			name:  "no config directory means the defaults",
			noDir: true,
			want:  "Config file: none\nNo config directory is set, so the defaults apply.\n",
		},
		{
			name:  "a broken file shows the line, the reason and the lines around it, and lists no unknown settings",
			setup: writeText("bogus: 1\nhooks:\n  ttl: 60\n timeout: 2\n"),
			want: file + `3 did not find expected key

  1 | bogus: 1
  2 | hooks:
> 3 |   ttl: 60
  4 |  timeout: 2
  5 |

The line is where the parser noticed the problem; the mistake may be just before it.
`,
			wantBad: true,
		},
		{
			name: "a directory is not a regular file",
			setup: func(t *testing.T, path string) {
				t.Helper()
				require.NoError(t, os.Mkdir(path, 0o700))
			},
			want:    file + "not a regular file\n",
			wantBad: true,
		},
		{
			name: "a FIFO is not a regular file, and is never opened",
			setup: func(t *testing.T, path string) {
				t.Helper()
				if err := syscall.Mkfifo(path, 0o600); err != nil {
					t.Skipf("mkfifo is unavailable: %v", err)
				}
			},
			want:    file + "not a regular file\n",
			wantBad: true,
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
			env := map[string]string{"HOME": home, "CLAUDE_CONFIG_DIR": home}
			if tt.noDir {
				env = nil
			}

			got, ok := statusline.ConfigCheck(func(k string) string { return env[k] })

			assert.Equal(t, tt.want, got)
			assert.Equal(t, !tt.wantBad, ok)
		})
	}
}

func writeText(text string) func(t *testing.T, path string) {
	return func(t *testing.T, path string) {
		t.Helper()
		require.NoError(t, os.WriteFile(path, []byte(text), 0o600))
	}
}

func TestConfigCheckPluginOptions(t *testing.T) {
	t.Parallel()

	const enabled = `{"enabledPlugins": {"statusline@m": true}, "pluginConfigs": {"statusline@m": {"options": {"theme": "saber", "path_emoji": "on"}}}}`

	tests := []struct {
		name     string
		settings string
		want     string
	}{
		{"no settings file is silent", "", ""},
		{"settings without options are silent", `{"model": "opus"}`, ""},
		{
			name:     "options are reported with where each setting came from",
			settings: enabled,
			want: `
Plugin options: ~/settings.json (statusline@m)
  theme: plugin options
  segments: default
  separator: config.yaml
  icons: default
  path.strip: default
  path.worktrees: default
  path.branch: default
  path.emoji: plugin options
  model.effort: default
  snapshot.enabled: default
  snapshot.path: default
`,
		},
		{
			name:     "malformed settings are reported and ignored",
			settings: `{"pluginConfigs": `,
			want: `
Plugin options: ~/settings.json
  not usable as settings: unexpected end of JSON input
  Ignored, so the config file and the defaults apply.
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			home := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(home, "claude-statusline"), 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(home, "claude-statusline", "config.yaml"), []byte("separator: \"|\"\n"), 0o600))
			if tt.settings != "" {
				require.NoError(t, os.WriteFile(filepath.Join(home, "settings.json"), []byte(tt.settings), 0o600))
			}
			env := map[string]string{"HOME": home, "CLAUDE_CONFIG_DIR": home}

			got, ok := statusline.ConfigCheck(func(k string) string { return env[k] })

			assert.True(t, ok)
			assert.Equal(t, "Config file: ~/claude-statusline/config.yaml\nParsed without errors.\n"+tt.want, got)
		})
	}
}

func TestConfigCheckIconsValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  string
		want    string
		notWant string
	}{
		{name: "an unknown set is listed", config: "icons: emoji\n", want: "\nInvalid values (ignored):\n  icons: emoji (use none or nerd)\n"},
		{name: "a boolean is listed as written", config: "icons: true\n", want: "  icons: true (use none or nerd)"},
		{name: "a control character is cleaned", config: "icons: \"a\\tb\"\n", want: "  icons: ab (use none or nerd)"},
		{name: "a trailing newline is listed", config: "icons: |\n  nerd\n", want: "  icons: nerd (use none or nerd)"},
		{name: "a leading tab is listed", config: "icons: \"\\tnerd\"\n", want: "Invalid values"},
		{name: "nerd is accepted", config: "icons: nerd\n", notWant: "Invalid values"},
		{name: "a different case is accepted", config: "icons: NERD\n", notWant: "Invalid values"},
		{name: "none is accepted", config: "icons: none\n", notWant: "Invalid values"},
		{name: "an absent key is accepted", config: "segments: [model]\n", notWant: "Invalid values"},
		{name: "a null is accepted", config: "icons:\n", notWant: "Invalid values"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			home := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(home, "claude-statusline"), 0o700))
			writeText(tt.config)(t, filepath.Join(home, "claude-statusline", "config.yaml"))
			env := map[string]string{"HOME": home, "CLAUDE_CONFIG_DIR": home}

			got, ok := statusline.ConfigCheck(func(k string) string { return env[k] })

			assert.True(t, ok)
			if tt.want != "" {
				assert.Contains(t, got, tt.want)
			}
			if tt.notWant != "" {
				assert.NotContains(t, got, tt.notWant)
			}
		})
	}
}

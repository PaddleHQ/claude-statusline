package cli_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/cli"
	"github.com/PaddleHQ/claude-statusline/internal/statusline"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noEnv(string) string { return "" }

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"prints the display name", `{"model":{"id":"claude-opus-4-7","display_name":"Opus 4.7"}}`, "Opus 4.7"},
		{"appends the effort level", `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"}}`, "Opus 4.7 [high]"},
		{"ignores unknown fields", `{"extra":[1,2],"model":{"id":"x","display_name":"Sonnet","other":true}}`, "Sonnet"},
		{"prints nothing when the model is missing", `{"effort":{"level":"high"}}`, ""},
		{"prints nothing for an empty display name", `{"model":{"display_name":""}}`, ""},
		{"prints nothing for invalid JSON", `{"model":`, ""},
		{"prints nothing for empty input", ``, ""},
		{"strips control characters", "{\"model\":{\"display_name\":\"Op\\u001b[31mus\\u0000\\n\\u007f 4\"}}", "Op[31mus 4"},
		{"prints no trailing newline", `{"model":{"display_name":"Haiku"}}`, "Haiku"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cli.Run(strings.NewReader(tt.input), &out, noEnv)

			assert.Equal(t, tt.want, out.String())
		})
	}
}

func TestExecute(t *testing.T) {
	t.Parallel()

	const payloadJSON = `{"model":{"display_name":"Opus"}}`

	tests := []struct {
		name  string
		args  []string
		build cli.Build
		want  string
	}{
		{"prints the version without a commit", []string{"--version"}, cli.Build{Version: "dev"}, "claude-statusline dev\n"},
		{"prints commit and date when set", []string{"--version"}, cli.Build{Version: "1.2.3", Commit: "abc1234", Date: "2026-10-13T10:00:00Z"}, "claude-statusline 1.2.3 (abc1234, 2026-10-13T10:00:00Z)\n"},
		{"only the first argument selects version", []string{"--version", "extra"}, cli.Build{Version: "1.2.3"}, "claude-statusline 1.2.3\n"},
		{"ignores version later in the arguments", []string{"other", "--version"}, cli.Build{Version: "1.2.3"}, "Opus"},
		{"prints the status line without arguments", nil, cli.Build{Version: "1.2.3"}, "Opus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cli.New(tt.build).Execute(tt.args, strings.NewReader(payloadJSON), &out, noEnv)

			assert.Equal(t, tt.want, out.String())
		})
	}
}

func TestEventFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"reads installed", []string{"--event=installed"}, "installed"},
		{"reads updated", []string{"--event=updated"}, "updated"},
		{"defaults to current when missing", nil, "current"},
		{"treats unknown values as current", []string{"--event=bogus"}, "current"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, cli.EventFlag(tt.args))
		})
	}
}

type failingReader struct{ t *testing.T }

func (r failingReader) Read([]byte) (int, error) {
	r.t.Error("the config command must not read stdin")
	return 0, io.ErrUnexpectedEOF
}

func TestExecuteConfig(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "claude-statusline"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(home, "claude-statusline", "config.yaml"), []byte("segments: [model]\n"), 0o600))
	busy := map[string]string{"CLAUDE_CONFIG_DIR": home, "HOME": home, "NO_COLOR": "1", "COLUMNS": "10"}

	run := func(args []string, env map[string]string) string {
		var out bytes.Buffer
		cli.New(cli.Build{Version: "dev"}).Execute(args, failingReader{t}, &out, func(k string) string { return env[k] })
		return out.String()
	}

	want := statusline.DefaultConfig()
	assert.Equal(t, want, run([]string{"config"}, busy))
	assert.Equal(t, want, run([]string{"config"}, nil))
	assert.Equal(t, want, run([]string{"config", "--x"}, busy))
}

func TestExecuteConfigCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config string
		want   string
		status int
	}{
		{
			name:   "a broken config exits 1 with the report",
			config: "hooks:\n  ttl: 60\n timeout: 2\n",
			want: `Config file: ~/claude-statusline/config.yaml
2 did not find expected key

  1 | hooks:
> 2 |   ttl: 60
  3 |  timeout: 2
  4 |

The line is where the parser noticed the problem; the mistake may be just before it.
`,
			status: 1,
		},
		{
			name:   "a good config exits 0",
			config: "segments: [model]\n",
			want:   "Config file: ~/claude-statusline/config.yaml\nParsed without errors.\n",
		},
		{
			name: "a missing config exits 0",
			want: "Config file: ~/claude-statusline/config.yaml\nDoes not exist, so the defaults apply.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			home := t.TempDir()
			if tt.config != "" {
				require.NoError(t, os.MkdirAll(filepath.Join(home, "claude-statusline"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(home, "claude-statusline", "config.yaml"), []byte(tt.config), 0o600))
			}
			env := map[string]string{"CLAUDE_CONFIG_DIR": home, "HOME": home}

			var out bytes.Buffer
			status := cli.New(cli.Build{Version: "dev"}).Execute([]string{"config-check"}, failingReader{t}, &out, func(k string) string { return env[k] })

			assert.Equal(t, tt.want, out.String())
			assert.Equal(t, tt.status, status)
		})
	}
}

func TestExecuteExitStatus(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "claude-statusline"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(home, "claude-statusline", "config.yaml"), []byte("segments: [model\n"), 0o600))
	env := map[string]string{"CLAUDE_CONFIG_DIR": home, "HOME": home, "NO_COLOR": "1"}

	for name, args := range map[string][]string{
		"version":            {"--version"},
		"help":               {"help"},
		"config":             {"config"},
		"subagent":           {"subagent"},
		"the status line":    nil,
		"an unknown command": {"bogus"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			status := cli.New(cli.Build{Version: "dev"}).Execute(args, strings.NewReader("{not json"), &out, func(k string) string { return env[k] })

			assert.Equal(t, 0, status)
		})
	}
}

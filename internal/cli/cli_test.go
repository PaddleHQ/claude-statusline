package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/cli"

	"github.com/stretchr/testify/assert"
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

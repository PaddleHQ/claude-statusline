package main

import (
	"bytes"
	"strings"
	"testing"

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
			run(strings.NewReader(tt.input), &out, noEnv)

			assert.Equal(t, tt.want, out.String())
		})
	}
}

func TestExecute(t *testing.T) {
	// Not parallel: it mutates the package-level build stamps.
	const payloadJSON = `{"model":{"display_name":"Opus"}}`

	tests := []struct {
		name    string
		args    []string
		version string
		commit  string
		date    string
		want    string
	}{
		{"prints the version without a commit", []string{"--version"}, "dev", "", "", "claude-statusline dev\n"},
		{"prints commit and date when set", []string{"--version"}, "1.2.3", "abc1234", "2026-10-13T10:00:00Z", "claude-statusline 1.2.3 (abc1234, 2026-10-13T10:00:00Z)\n"},
		{"only the first argument selects version", []string{"--version", "extra"}, "1.2.3", "", "", "claude-statusline 1.2.3\n"},
		{"ignores version later in the arguments", []string{"other", "--version"}, "1.2.3", "", "", "Opus"},
		{"prints the status line without arguments", nil, "1.2.3", "", "", "Opus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVersion, oldCommit, oldDate := version, commit, date
			t.Cleanup(func() { version, commit, date = oldVersion, oldCommit, oldDate })
			version, commit, date = tt.version, tt.commit, tt.date

			var out bytes.Buffer
			execute(tt.args, strings.NewReader(payloadJSON), &out, noEnv)

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
			assert.Equal(t, tt.want, eventFlag(tt.args))
		})
	}
}

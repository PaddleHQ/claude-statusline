package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"prints the display name", `{"model":{"id":"claude-opus-4-7","display_name":"Opus 4.7"}}`, "Opus 4.7"},
		{"ignores unknown fields", `{"effort":{"level":"high"},"extra":[1,2],"model":{"id":"x","display_name":"Sonnet","other":true}}`, "Sonnet"},
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
			run(strings.NewReader(tt.input), &out)

			if got := out.String(); got != tt.want {
				t.Errorf("run() printed %q, want %q", got, tt.want)
			}
		})
	}
}

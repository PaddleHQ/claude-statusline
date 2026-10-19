package statusline_test

import "testing"

// TestBuildInputAndColour pins what the line prints for empty input and for control characters, and when NO_COLOR removes the color.
func TestBuildInputAndColour(t *testing.T) {
	t.Parallel()

	tests := []scenario{
		{
			name:    `NO_COLOR=1 removes every color from the output`,
			session: `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}`,
			config:  `segments: [model, model]`,
			env:     map[string]string{`NO_COLOR`: `1`},
			want:    `Opus 4.7 [high] · Opus 4.7 [high]`,
		},
		{
			name:    `An empty NO_COLOR variable does not disable color; only a non-empty value does`,
			session: `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}`,
			config:  `segments: [model, model]`,
			env:     map[string]string{`NO_COLOR`: ``},
			want:    `Opus 4.7 [high] \e[2m·\e[0m Opus 4.7 [high]`,
		},
		{
			name:    `Empty input prints nothing`,
			session: ``,
			want:    ``,
		},
		{
			name:    `Control characters in the model name from the payload are stripped before display`,
			session: `{"model":{"display_name":"Op\u001b[31mus"},"session_id":"abc123"}`,
			want:    `Op[31mus`,
		},
	}
	runAll(t, tests)
}

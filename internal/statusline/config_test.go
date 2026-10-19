package statusline_test

import "testing"

// TestBuildConfigFile pins how the config file reaches the line: what it skips, what falls back and that no value is ever executed.
func TestBuildConfigFile(t *testing.T) {
	t.Parallel()

	const session = `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}`
	tests := []scenario{
		{
			name:    `Comments, blank lines and unknown keys are skipped while the known keys apply, and a section keeps its other defaults`,
			session: session,
			config: `# a comment

   # an indented comment
separator: "|"
segments: [model, model]
model:
  effort: false
  label: ignored
unknown:
  key: 1
unknown_top: 2`,
			want: `Opus 4.7 \e[2m|\e[0m Opus 4.7`,
		},
		{
			name:    `Invalid YAML ignores the whole file and the defaults apply`,
			session: session,
			config: `separator: "|"
segments: [model, model`,
			want: `Opus 4.7 [high]`,
		},
		{
			name:    `A duplicate key makes the YAML invalid, so the whole file is ignored`,
			session: session,
			config: `segments: [model, model]
segments: [model]`,
			want: `Opus 4.7 [high]`,
		},
		{
			name:    `A file that is not a mapping is ignored`,
			session: session,
			config:  `- segments: [model, model]`,
			want:    `Opus 4.7 [high]`,
		},
		{
			name:    `A value of the wrong type falls back to its default while the keys around it still apply`,
			session: session,
			config: `model:
  effort: maybe
separator: [a, b]
segments: [model, model]`,
			want: `Opus 4.7 [high] \e[2m·\e[0m Opus 4.7 [high]`,
		},
		{
			name:    `An empty or unquoted non-string value falls back to the default`,
			session: session,
			config: `separator: ""
theme: 5
segments: [model, model]`,
			want: `Opus 4.7 [high] \e[2m·\e[0m Opus 4.7 [high]`,
		},
		{
			name:    `A string with a control character falls back to the default`,
			session: session,
			config: `separator: "a\nb"
segments: [model, model]`,
			want: `Opus 4.7 [high] \e[2m·\e[0m Opus 4.7 [high]`,
		},
		{
			name:    `A segments entry of the wrong type makes the whole list fall back to the default`,
			session: session,
			config:  `segments: [model, 5]`,
			want:    `Opus 4.7 [high]`,
		},
		{
			name:    `The defaults apply when there is no config file`,
			session: session,
			want:    `Opus 4.7 [high]`,
		},
		{
			name:    `Unknown, malformed and path-like segment names are dropped, and an unknown theme falls back to the default`,
			session: session,
			config: `segments: [model, bogus, Bad-Name, ../x, model]
theme: nonexistent`,
			want: `Opus 4.7 [high] \e[2m·\e[0m Opus 4.7 [high]`,
		},
		{
			name:    `A command substitution in the separator setting is shown as text and never executed`,
			session: session,
			config: `segments: [model, model]
separator: "$(touch @TMP@/pwned)"`,
			want: `Opus 4.7 [high] \e[2m$(touch @TMP@/pwned)\e[0m Opus 4.7 [high]`,
		},
	}
	runAll(t, tests)
}

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
			name:    `Invalid YAML ignores the whole file, the defaults apply and an error row comes first`,
			session: session,
			config: `separator: "|"
segments: [model, model`,
			want: "\\e[31m~/claude-statusline/config.yaml:1 did not find expected ',' or ']' · run /statusline:config-check\\e[0m\nOpus 4.7 [high]",
		},
		{
			name:    `A duplicate key makes the YAML invalid, so the whole file is ignored and an error row comes first`,
			session: session,
			config: `segments: [model, model]
segments: [model]`,
			want: "\\e[31m~/claude-statusline/config.yaml:2 mapping key \"segments\" already defined at line 1 · run /statusline:config-check\\e[0m\nOpus 4.7 [high]",
		},
		{
			name:    `A file that is not a mapping is ignored and an error row comes first`,
			session: session,
			config:  `- segments: [model, model]`,
			want:    "\\e[31m~/claude-statusline/config.yaml: the top level must be a mapping · run /statusline:config-check\\e[0m\nOpus 4.7 [high]",
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
			name:    `A negative or fractional number falls back to the default`,
			session: session,
			config: `rate5h:
  width: -3
  warn: 2.5
segments: [model, model]`,
			want: `Opus 4.7 [high] \e[2m·\e[0m Opus 4.7 [high]`,
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
		{
			name:    `Injection attempts in width and label settings are shown as text and never executed`,
			session: `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}`,
			config: `rate5h:
  width: "5; touch @TMP@/pwned"
rate7d:
  label: "` + "`touch @TMP@/pwned`" + `"`,
			want: "Opus 4.7 [high] \\e[2m·\\e[0m 5h: \\e[32m█████████\\e[0m\\e[2m░\\e[0m\\e[1m┃\\e[0m\\e[2m░░░░░░░░░\\e[0m 45% \\e[2m·\\e[0m `touch @TMP@/pwned`: \\e[32m████████\\e[0m\\e[2m░░\\e[0m\\e[1m┃\\e[0m\\e[2m░░░░░░░░░\\e[0m 40%",
		},
	}
	runAll(t, tests)
}

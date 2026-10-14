package statusline_test

import "testing"

// TestBuildConfigFile pins how the config file reaches the line: what it skips, what falls back and that no value is ever executed.
func TestBuildConfigFile(t *testing.T) {
	t.Parallel()

	tests := []scenario{
		{
			name:    `Config parsing skips comments, blank lines, junk and unknown keys, ignores spacing around =, and lets the last duplicate win`,
			session: `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}`,
			config: `# a comment

   separator   =   |
segments=model model
model.effort = true
model.effort = false
model.effort = true
model.label = ignored
not a setting line
unknown.key = 1`,
			want: `Opus 4.7 [high] \e[2m|\e[0m Opus 4.7 [high]`,
		},
		{
			name:    `Malformed config keys are ignored while the valid ones around them still apply`,
			session: `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}`,
			config: `model.effort.x = false
Model.effort = false
model-effort = false
model..effort = false
.effort = false
model. = false
segments.x = nothing
separator = ok
segments = model model`,
			want: `Opus 4.7 [high] \e[2mok\e[0m Opus 4.7 [high]`,
		},
		{
			name:    `Invalid or empty config values fall back to the defaults`,
			session: `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}`,
			config: `model.effort = maybe
separator =
segments = model model`,
			want: `Opus 4.7 [high] \e[2m·\e[0m Opus 4.7 [high]`,
		},
		{
			name:    `The defaults apply when there is no config file`,
			session: `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}`,
			want:    `Opus 4.7 [high]`,
		},
		{
			name:    `Unknown, malformed and path-like segment names are dropped, and an unknown theme falls back to the default`,
			session: `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}`,
			config: `segments = model bogus Bad-Name ../x model
theme = nonexistent`,
			want: `Opus 4.7 [high] \e[2m·\e[0m Opus 4.7 [high]`,
		},
		{
			name:    `A command substitution in the separator setting is shown as text and never executed`,
			session: `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"session_id":"abc123"}`,
			config: `segments = model model
separator = $(touch @TMP@/pwned)`,
			want: `Opus 4.7 [high] \e[2m$(touch @TMP@/pwned)\e[0m Opus 4.7 [high]`,
		},
		{
			name:    `Injection attempts in width and label settings are shown as text and never executed`,
			session: `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}`,
			config:  "rate5h.width = 5; touch @TMP@/pwned\nrate7d.label = `touch @TMP@/pwned`",
			want:    "Opus 4.7 [high] \\e[2m·\\e[0m 5h: \\e[32m█████████\\e[0m\\e[2m░\\e[0m\\e[1m┃\\e[0m\\e[2m░░░░░░░░░\\e[0m 45% \\e[2m·\\e[0m `touch @TMP@/pwned`: \\e[32m████████\\e[0m\\e[2m░░\\e[0m\\e[1m┃\\e[0m\\e[2m░░░░░░░░░\\e[0m 40%",
		},
	}
	runAll(t, tests)
}

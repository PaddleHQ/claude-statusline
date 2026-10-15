package statusline_test

import "testing"

// TestBuildThemes pins where the theme setting is looked up, and that a theme that cannot be used leaves the built-in look.
func TestBuildThemes(t *testing.T) {
	t.Parallel()

	tests := []scenario{
		{
			name:    `A theme file with an unknown color name is rejected and the built-in theme is used`,
			session: `{"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}`,
			config: `theme = ~/bad.yaml
segments = rate5h`,
			files: map[string]string{
				`bad.yaml`: `colors:
  text: ""
  muted: dim
  good: chartreuse
  warn: yellow
  bad: red
  marker: bold
  path: bold 33
bar:
  style: gradient
  full: "▰"
  partials: ["▏", "▎", "▍", "▌", "▋", "▊", "▉"]
  empty: "▱"
  marker: "/"
  ramp: ["#94e2ad", "#f9e2af", "#f38ba8"]`,
			},
			want: `5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60%`,
		},
		{
			name:    `A theme file whose gradient ramp is invalid is rejected and the built-in theme is used`,
			session: `{"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}`,
			config: `theme = ~/bad.yaml
segments = rate5h`,
			files: map[string]string{
				`bad.yaml`: `colors:
  text: ""
  muted: dim
  good: green
  warn: yellow
  bad: red
  marker: bold
  path: bold 33
bar:
  style: gradient
  full: "▰"
  partials: ["▏", "▎", "▍", "▌", "▋", "▊", "▉"]
  empty: "▱"
  marker: "/"
  ramp: ["#fff", "#000000", "#000000"]`,
			},
			want: `5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60%`,
		},
		{
			name:    `A theme file that is not valid YAML is rejected and the built-in theme is used`,
			session: `{"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}`,
			config: `theme = ~/bad.yaml
segments = rate5h`,
			files: map[string]string{
				`bad.yaml`: `colors: [unclosed`,
			},
			want: `5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60%`,
		},
		{
			name:    `theme = default is accepted and draws the built-in look`,
			session: `{"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}`,
			config: `theme = default
segments = rate5h`,
			want: `5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60%`,
		},
		{
			name:    `A theme file named by an absolute path is loaded`,
			session: `{"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}`,
			config: `theme = @TMP@/claude-statusline/ocean.yaml
segments = rate5h`,
			files: map[string]string{
				`claude-statusline/ocean.yaml`: `colors:
  text: ""
  muted: bright-black
  good: cyan
  warn: ""
  bad: ""
  marker: ""
  path: ""
bar:
  style: solid
  full: "#"
  empty: "."
  marker: "|"
  partials: ["1", "2", "3", "4", "5", "6", "7"]`,
			},
			want: `5h: \e[36m##########\e[0m|\e[36m#\e[0m\e[90m........\e[0m 60%`,
		},
		{
			name:    `A relative theme path is resolved against the config directory`,
			session: `{"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}`,
			config: `theme = themes/ocean.yaml
segments = rate5h`,
			files: map[string]string{
				`claude-statusline/themes/ocean.yaml`: `colors:
  text: ""
  muted: bright-black
  good: cyan
  warn: ""
  bad: ""
  marker: ""
  path: ""
bar:
  style: solid
  full: "#"
  empty: "."
  marker: "|"
  partials: ["1", "2", "3", "4", "5", "6", "7"]`,
			},
			want: `5h: \e[36m##########\e[0m|\e[36m#\e[0m\e[90m........\e[0m 60%`,
		},
		{
			name:    `A theme file that does not exist falls back to the built-in theme`,
			session: `{"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}`,
			config: `theme = ~/nope.yaml
segments = rate5h`,
			want: `5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60%`,
		},
		{
			name:    `A theme file with an unknown key is rejected and the built-in theme is used`,
			session: `{"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}`,
			config: `theme = ~/bad.yaml
segments = rate5h`,
			files: map[string]string{
				`bad.yaml`: `colrs:
  text: ""
  muted: dim
  good: green
  warn: yellow
  bad: red
  marker: bold
  path: bold 33
bar:
  style: gradient
  full: "▰"
  partials: ["▏", "▎", "▍", "▌", "▋", "▊", "▉"]
  empty: "▱"
  marker: "/"
  ramp: ["#94e2ad", "#f9e2af", "#f38ba8"]`,
			},
			want: `5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60%`,
		},
	}
	runAll(t, tests)
}

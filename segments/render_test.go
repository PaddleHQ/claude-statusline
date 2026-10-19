package segments_test

import (
	"strings"
	"testing"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const drawNow = 1760000000

type drawCase struct {
	name    string
	segs    string
	cfg     string
	session string
	cols    int
	theme   string
	plain   bool
	want    string
}

// escapes writes an escape byte as \e, the way the feature files do.
func escapes(s string) string { return strings.ReplaceAll(s, "\x1b", `\e`) }

// draw renders the named segments of a session through the real adapters, theme and bar drawing, joined by the separator.
func draw(t *testing.T, c *drawCase) string {
	t.Helper()

	th := theme.Load(c.theme, "", "")
	if c.plain {
		th = th.WithoutColor()
	}
	p := payload.Parse([]byte(c.session))
	incoming := ratecache.FromPayload(p.RateLimits)
	limits := incoming.Visible(drawNow)
	env := segments.NewEnv(segments.Env{
		Payload:    p,
		Config:     config.New(c.cfg, segments.Defaults()...),
		Theme:      th,
		Now:        time.Unix(drawNow, 0),
		Columns:    c.cols,
		Home:       "/home/me",
		RateLimits: func() ratecache.Limits { return limits },
	})
	var parts []string
	for name := range strings.FieldsSeq(c.segs) {
		seg, _, ok := env.Lookup(name)
		require.True(t, ok, name)
		if text := seg.Render(); text != "" {
			parts = append(parts, text)
		}
	}
	return escapes(strings.Join(parts, env.Separator()))
}

// TestSegmentsDrawnWithTheRealTheme pins the exact text and colors each segment draws once the program's own adapters,
// the default settings, a theme and the bar drawing stand behind it.
func TestSegmentsDrawnWithTheRealTheme(t *testing.T) {
	t.Parallel()

	tests := []drawCase{
		{
			name: `Usage of 60% with half the window elapsed is drawn in green`, segs: `model rate5h`, cfg: "", session: `{"model":{"display_name":"Opus 4.7"},"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}`, cols: 200, theme: `classic`, plain: false,
			want: `Opus 4.7 \e[2m·\e[0m 5h: \e[32m██████████\e[0m\e[1m┃\e[0m\e[32m█\e[0m\e[2m░░░░░░░░\e[0m 60%`,
		},
		{
			name: `Usage of 75% with half the window elapsed is drawn in yellow`, segs: `model rate5h`, cfg: "", session: `{"model":{"display_name":"Opus 4.7"},"rate_limits":{"five_hour":{"used_percentage":75,"resets_at":1760009000}},"session_id":"abc123"}`, cols: 200, theme: `classic`, plain: false,
			want: `Opus 4.7 \e[2m·\e[0m 5h: \e[33m██████████\e[0m\e[1m┃\e[0m\e[33m████\e[0m\e[2m░░░░░\e[0m 75%`,
		},
		{
			name: `Usage of 92% with half the window elapsed is drawn in red`, segs: `model rate5h`, cfg: "", session: `{"model":{"display_name":"Opus 4.7"},"rate_limits":{"five_hour":{"used_percentage":92,"resets_at":1760009000}},"session_id":"abc123"}`, cols: 200, theme: `classic`, plain: false,
			want: `Opus 4.7 \e[2m·\e[0m 5h: \e[31m██████████\e[0m\e[1m┃\e[0m\e[31m███████▍\e[0m\e[2m░\e[0m 92%`,
		},
		{
			name: `Usage of 5% with half the window elapsed is drawn in red, with the pace marker well ahead of it`, segs: `model rate5h`, cfg: "", session: `{"model":{"display_name":"Opus 4.7"},"rate_limits":{"five_hour":{"used_percentage":5,"resets_at":1760009000}},"session_id":"abc123"}`, cols: 200, theme: `classic`, plain: false,
			want: `Opus 4.7 \e[2m·\e[0m 5h: \e[31m█\e[0m\e[2m░░░░░░░░░\e[0m\e[1m┃\e[0m\e[2m░░░░░░░░░\e[0m 5%`,
		},
		{
			name: `Usage of 96% fills the bar and carries on past the pace marker`, segs: `model rate5h`, cfg: "", session: `{"model":{"display_name":"Opus 4.7"},"rate_limits":{"five_hour":{"used_percentage":96,"resets_at":1760001800}},"session_id":"abc123"}`, cols: 200, theme: `classic`, plain: false,
			want: `Opus 4.7 \e[2m·\e[0m 5h: \e[31m██████████████████\e[0m\e[1m┃\e[0m\e[31m▎\e[0m 96%`,
		},
		{
			name: `At 100 columns the usage bars use their narrow width`, segs: `model rate5h rate7d`, cfg: "", session: `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}`, cols: 100, theme: `classic`, plain: false,
			want: `Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m████▌\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%`,
		},
		{
			name: `At 140 columns the usage bars use their medium width`, segs: `model rate5h rate7d`, cfg: "", session: `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}`, cols: 140, theme: `classic`, plain: false,
			want: `Opus 4.7 [high] \e[2m·\e[0m 5h: \e[32m██████▎\e[0m\e[1m┃\e[0m\e[2m░░░░░░\e[0m 45% \e[2m·\e[0m 7d: \e[32m█████▋\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░░░\e[0m 40%`,
		},
		{
			name: `A 10 column terminal uses the narrow bar width from rate5h.width_narrow`, segs: `rate5h rate7d`, cfg: `rate5h: {width_narrow: 3}`, session: `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}`, cols: 10, theme: `classic`, plain: false,
			want: `5h: \e[32m█▊\e[0m\e[1m┃\e[0m\e[2m░\e[0m 45% \e[2m·\e[0m 7d: \e[32m████\e[0m\e[2m░\e[0m\e[1m┃\e[0m\e[2m░░░░\e[0m 40%`,
		},
		{
			name: `Context use of 30% is drawn in green`, segs: `ctx`, cfg: "", session: `{"context_window":{"used_percentage":30}}`, cols: 200, theme: `classic`, plain: false,
			want: `ctx: \e[32m███\e[0m\e[2m░░░░░░░\e[0m 30%`,
		},
		{
			name: `Context use of 50% is drawn in yellow`, segs: `ctx`, cfg: "", session: `{"context_window":{"used_percentage":50}}`, cols: 200, theme: `classic`, plain: false,
			want: `ctx: \e[33m█████\e[0m\e[2m░░░░░\e[0m 50%`,
		},
		{
			name: `Context use of 90% is drawn in red`, segs: `ctx`, cfg: "", session: `{"context_window":{"used_percentage":90}}`, cols: 200, theme: `classic`, plain: false,
			want: `ctx: \e[31m█████████\e[0m\e[2m░\e[0m 90%`,
		},
		{
			name: `At 100 columns the context bar is narrower and a fractional percentage is rounded`, segs: `ctx`, cfg: "", session: `{"context_window":{"used_percentage":64.6}}`, cols: 100, theme: `classic`, plain: true,
			want: `ctx: ███▉░░ 65%`,
		},
		{
			name: `Token counts below a thousand are shown as they are, and exactly a thousand becomes 1.0k`, segs: `tokens`, cfg: "", session: `{"context_window":{"total_input_tokens":999,"total_output_tokens":1000}}`, cols: 200, theme: `classic`, plain: true,
			want: `↓ 999 · ↑ 1.0k`,
		},
		{
			name: `Token counts in the thousands are shown with a k suffix`, segs: `tokens`, cfg: "", session: `{"context_window":{"total_input_tokens":1500,"total_output_tokens":999999}}`, cols: 200, theme: `classic`, plain: true,
			want: `↓ 1.5k · ↑ 999.9k`,
		},
		{
			name: `Token counts in the millions are shown with an M suffix`, segs: `tokens`, cfg: "", session: `{"context_window":{"total_input_tokens":1000000,"total_output_tokens":1500000}}`, cols: 200, theme: `classic`, plain: true,
			want: `↓ 1.0M · ↑ 1.5M`,
		},
		{
			name: `The tokens segment with color on, with a dimmed separator between the input and output counts`, segs: `tokens`, cfg: "", session: `{"context_window":{"total_input_tokens":1500,"total_output_tokens":250}}`, cols: 200, theme: `classic`, plain: false,
			want: `↓ 1.5k \e[2m·\e[0m ↑ 250`,
		},
		{
			name: `A cache hit rate of 80% is shown without color`, segs: `cache`, cfg: "", session: `{"context_window":{"current_usage":{"input_tokens":10,"cache_creation_input_tokens":10,"cache_read_input_tokens":80}}}`, cols: 200, theme: `classic`, plain: false,
			want: `⧉ 80%`,
		},
		{
			name: `A cache hit rate of 5% is shown without color`, segs: `cache`, cfg: "", session: `{"context_window":{"current_usage":{"input_tokens":95,"cache_creation_input_tokens":0,"cache_read_input_tokens":5}}}`, cols: 200, theme: `classic`, plain: false,
			want: `⧉ 5%`,
		},
		{
			name: `A cache hit rate of 3% is shown in red`, segs: `cache`, cfg: "", session: `{"context_window":{"current_usage":{"input_tokens":50,"cache_creation_input_tokens":50,"cache_read_input_tokens":4}}}`, cols: 200, theme: `classic`, plain: false,
			want: `\e[31m⧉ 3%\e[0m`,
		},
		{
			name: `cache.warn = 90 turns an 80% hit rate red`, segs: `cache`, cfg: `cache: {warn: 90}`, session: `{"context_window":{"current_usage":{"input_tokens":10,"cache_creation_input_tokens":10,"cache_read_input_tokens":80}}}`, cols: 200, theme: `classic`, plain: false,
			want: `\e[31m⧉ 80%\e[0m`,
		},
		{
			name: `The cache segment is hidden when the token counts add up to zero, so there is no rate to compute`, segs: `cache`, cfg: "", session: `{"context_window":{"current_usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`, cols: 200, theme: `classic`, plain: true,
			want: ``,
		},
		{
			name: `The gradient theme at 95% context use`, segs: `ctx`, cfg: "", session: `{"context_window":{"used_percentage":95}}`, cols: 200, theme: `gradient`, plain: false,
			want: `ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[38;2;249;217;175m▰\e[0m\e[38;2;248;198;173m▰\e[0m\e[38;2;246;178;172m▰\e[0m\e[38;2;245;159;170m▰\e[0m\e[38;2;243;139;168m▰\e[0m \e[31m95%\e[0m`,
		},
		{
			name: `The gradient theme at 0% context use draws an entirely empty bar`, segs: `ctx`, cfg: "", session: `{"context_window":{"used_percentage":0}}`, cols: 200, theme: `gradient`, plain: false,
			want: `ctx: \e[38;2;74;74;74m▱▱▱▱▱▱▱▱▱▱\e[0m \e[32m0%\e[0m`,
		},
		{
			name: `The gradient theme's context bar at 100 columns`, segs: `ctx`, cfg: "", session: `{"context_window":{"used_percentage":71}}`, cols: 100, theme: `gradient`, plain: false,
			want: `ctx: \e[38;2;148;226;173m▰\e[0m\e[38;2;188;226;173m▰\e[0m\e[38;2;228;226;174m▰\e[0m\e[38;2;248;209;174m▰\e[0m\e[38;2;74;74;74m▱▱\e[0m \e[33m71%\e[0m`,
		},
		{
			name: `The gradient theme's rate-limit bars at 140 columns, with a pace marker slash`, segs: `rate5h rate7d`, cfg: "", session: `{"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}`, cols: 140, theme: `gradient`, plain: false,
			want: `5h: \e[38;2;148;226;173m▰\e[0m\e[38;2;163;226;173m▰\e[0m\e[38;2;179;226;173m▰\e[0m\e[38;2;194;226;173m▰\e[0m\e[38;2;210;226;174m▰\e[0m\e[38;2;225;226;174m▰\e[0m\e[38;2;74;74;74m▱\e[0m\e[1m/\e[0m\e[38;2;74;74;74m▱▱▱▱▱▱\e[0m \e[32m45%\e[0m \e[2m·\e[0m 7d: \e[38;2;148;226;173m▰\e[0m\e[38;2;163;226;173m▰\e[0m\e[38;2;179;226;173m▰\e[0m\e[38;2;194;226;173m▰\e[0m\e[38;2;210;226;174m▰\e[0m\e[38;2;225;226;174m▰\e[0m\e[38;2;74;74;74m▱\e[0m\e[1m/\e[0m\e[38;2;74;74;74m▱▱▱▱▱▱\e[0m \e[32m40%\e[0m`,
		},
		{
			name: `The gradient theme's rate-limit bars at 100 columns`, segs: `rate5h rate7d`, cfg: "", session: `{"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000},"seven_day":{"used_percentage":40,"resets_at":1760302400}},"session_id":"abc123"}`, cols: 100, theme: `gradient`, plain: false,
			want: `5h: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;237;226;174m▰\e[0m\e[1m/\e[0m\e[38;2;74;74;74m▱▱▱▱\e[0m \e[32m45%\e[0m \e[2m·\e[0m 7d: \e[38;2;148;226;173m▰\e[0m\e[38;2;170;226;173m▰\e[0m\e[38;2;192;226;173m▰\e[0m\e[38;2;215;226;174m▰\e[0m\e[38;2;74;74;74m▱\e[0m\e[1m/\e[0m\e[38;2;74;74;74m▱▱▱▱\e[0m \e[32m40%\e[0m`,
		},
		{
			name: `The gradient theme with NO_COLOR keeps the block characters but drops every color`, segs: `ctx rate5h`, cfg: "", session: `{"rate_limits":{"five_hour":{"used_percentage":75,"resets_at":1760009000}},"context_window":{"used_percentage":62}}`, cols: 200, theme: `gradient`, plain: true,
			want: `ctx: ▰▰▰▰▰▰▱▱▱▱ 62% · 5h: ▰▰▰▰▰▰▰▰▰▰/▰▰▰▰▱▱▱▱▱ 75%`,
		},
		{
			name: `The gradient theme at 92% usage with half the window elapsed`, segs: `rate5h`, cfg: "", session: `{"rate_limits":{"five_hour":{"used_percentage":92,"resets_at":1760009000}},"session_id":"abc123"}`, cols: 200, theme: `gradient`, plain: false,
			want: `5h: \e[38;2;148;226;173m▰\e[0m\e[38;2;158;226;173m▰\e[0m\e[38;2;169;226;173m▰\e[0m\e[38;2;179;226;173m▰\e[0m\e[38;2;190;226;173m▰\e[0m\e[38;2;201;226;174m▰\e[0m\e[38;2;211;226;174m▰\e[0m\e[38;2;222;226;174m▰\e[0m\e[38;2;233;226;174m▰\e[0m\e[38;2;243;226;174m▰\e[0m\e[1m/\e[0m\e[38;2;249;213;174m▰\e[0m\e[38;2;248;204;174m▰\e[0m\e[38;2;247;194;173m▰\e[0m\e[38;2;247;185;172m▰\e[0m\e[38;2;246;176;171m▰\e[0m\e[38;2;245;167;171m▰\e[0m\e[38;2;245;158;170m▰\e[0m\e[38;2;74;74;74m▱▱\e[0m \e[31m92%\e[0m`,
		},
		{
			name: `The gradient theme at 60% usage with half the window elapsed`, segs: `rate5h`, cfg: "", session: `{"rate_limits":{"five_hour":{"used_percentage":60,"resets_at":1760009000}},"session_id":"abc123"}`, cols: 200, theme: `gradient`, plain: false,
			want: `5h: \e[38;2;148;226;173m▰\e[0m\e[38;2;158;226;173m▰\e[0m\e[38;2;169;226;173m▰\e[0m\e[38;2;179;226;173m▰\e[0m\e[38;2;190;226;173m▰\e[0m\e[38;2;201;226;174m▰\e[0m\e[38;2;211;226;174m▰\e[0m\e[38;2;222;226;174m▰\e[0m\e[38;2;233;226;174m▰\e[0m\e[38;2;243;226;174m▰\e[0m\e[1m/\e[0m\e[38;2;249;213;174m▰\e[0m\e[38;2;74;74;74m▱▱▱▱▱▱▱▱\e[0m \e[32m60%\e[0m`,
		},
		{
			name: `The gradient theme at 75% usage with half the window elapsed`, segs: `rate5h`, cfg: "", session: `{"rate_limits":{"five_hour":{"used_percentage":75,"resets_at":1760009000}},"session_id":"abc123"}`, cols: 200, theme: `gradient`, plain: false,
			want: `5h: \e[38;2;148;226;173m▰\e[0m\e[38;2;158;226;173m▰\e[0m\e[38;2;169;226;173m▰\e[0m\e[38;2;179;226;173m▰\e[0m\e[38;2;190;226;173m▰\e[0m\e[38;2;201;226;174m▰\e[0m\e[38;2;211;226;174m▰\e[0m\e[38;2;222;226;174m▰\e[0m\e[38;2;233;226;174m▰\e[0m\e[38;2;243;226;174m▰\e[0m\e[1m/\e[0m\e[38;2;249;213;174m▰\e[0m\e[38;2;248;204;174m▰\e[0m\e[38;2;247;194;173m▰\e[0m\e[38;2;247;185;172m▰\e[0m\e[38;2;74;74;74m▱▱▱▱▱\e[0m \e[33m75%\e[0m`,
		},
		{
			name: `The path segment is drawn in bold blue`, segs: `path`, cfg: "", session: `{"cwd":"/home/me/work/app"}`, cols: 200, theme: `classic`, plain: false,
			want: `\e[1m\e[38;5;33m~/work/app\e[0m`,
		},
		{
			name: `A payload with only an output token count still shows the tokens segment`, segs: `tokens`, cfg: "", session: `{"context_window":{"total_output_tokens":0}}`, cols: 200, theme: `classic`, plain: true,
			want: `↑ 0`,
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, draw(t, tt))
		})
	}
}

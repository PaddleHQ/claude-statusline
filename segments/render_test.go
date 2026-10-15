package segments_test

import (
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	env := &segments.Env{
		Payload: payload.Parse([]byte(c.session)),
		Config:  config.New(c.cfg, segments.Defaults()...),
		Theme:   th,
		Columns: c.cols,
		Home:    "/home/me",
	}
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
			name: `cache.warn = 90 turns an 80% hit rate red`, segs: `cache`, cfg: `cache.warn = 90`, session: `{"context_window":{"current_usage":{"input_tokens":10,"cache_creation_input_tokens":10,"cache_read_input_tokens":80}}}`, cols: 200, theme: `classic`, plain: false,
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

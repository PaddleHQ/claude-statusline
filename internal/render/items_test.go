package render_test

import (
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/hook"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/render"
	"github.com/PaddleHQ/claude-statusline/internal/theme"

	"github.com/stretchr/testify/assert"
)

type stub struct {
	name  string
	group layout.Group
	text  string
}

func (s *stub) Render() string { return s.text }

func lookupIn(stubs ...*stub) func(string) (render.Segment, layout.Group, bool) {
	return func(name string) (render.Segment, layout.Group, bool) {
		for _, s := range stubs {
			if s.name == name {
				return s, s.group, true
			}
		}
		return nil, 0, false
	}
}

func standardStubs() []*stub {
	return []*stub{
		{"path", layout.Identity, "/srv/app"},
		{"model", layout.Identity, "Opus"},
		{"ctx", layout.Identity, "ctx: 42%"},
		{"tokens", layout.Usage, "↓ 1.5k · ↑ 2.0k"},
		{"rate5h", layout.Rates, "5h: 45%"},
		{"empty", layout.Usage, ""},
	}
}

func newLine(stubs []*stub, cfg string, th theme.Theme, columns int, hooks []hook.Result) *render.Line {
	c := config.New(cfg)
	return &render.Line{
		Names:     strings.Fields(c.String("segments")),
		Lookup:    lookupIn(stubs...),
		Columns:   columns,
		Separator: " " + th.Paint(th.Muted, c.String("separator")) + " ",
		Anchor:    th.Paint(th.Muted, "↳"),
		Hooks:     hooks,
	}
}

func plainClassic() theme.Theme {
	classic := theme.Load("classic", "", "")
	return classic.WithoutColor()
}

func hookResult(name string, slot hook.Slot, text string) hook.Result {
	return hook.Result{File: name + ".sh", Name: name, Slot: slot, Text: text}
}

func TestRender(t *testing.T) {
	t.Parallel()

	th := theme.Load("classic", "", "")
	tests := []struct {
		name string
		cfg  string
		th   theme.Theme
		want string
	}{
		{"default segments", "", th, "/srv/app \033[2m·\033[0m Opus \033[2m·\033[0m ctx: 42% \033[2m·\033[0m ↓ 1.5k · ↑ 2.0k \033[2m·\033[0m 5h: 45%"},
		{"unknown names are skipped", "segments = bogus model Bad-Name ../x", th, "Opus"},
		{"repeated segment is joined with the painted separator", "segments = model model", th, "Opus \033[2m·\033[0m Opus"},
		{"custom separator is painted muted", "segments = model model\nseparator = |", th, "Opus \033[2m|\033[0m Opus"},
		{"no color leaves the separator bare", "segments = model model", th.WithoutColor(), "Opus · Opus"},
		{"only unknown segments render nothing", "segments = bogus", th, ""},
		{"empty segments falls back to the default", "segments =", th, "/srv/app \033[2m·\033[0m Opus \033[2m·\033[0m ctx: 42% \033[2m·\033[0m ↓ 1.5k · ↑ 2.0k \033[2m·\033[0m 5h: 45%"},
		{"separator injection stays data", "segments = model model\nseparator = $(touch x)", th, "Opus \033[2m$(touch x)\033[0m Opus"},
		{"segments with nothing to show are dropped", "segments = model empty model", th, "Opus \033[2m·\033[0m Opus"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, newLine(standardStubs(), tt.cfg, tt.th, 0, nil).Render())
		})
	}
}

func TestRenderPlacesHooks(t *testing.T) {
	t.Parallel()

	th := plainClassic()
	tests := []struct {
		name  string
		cfg   string
		hooks []hook.Result
		want  string
	}{
		{
			"ctx slot goes after the context segment",
			"segments = path model ctx tokens",
			[]hook.Result{hookResult("a", hook.SlotCtx, "A")},
			"/srv/app · Opus · ctx: 42% · A · ↓ 1.5k · ↑ 2.0k",
		},
		{
			"ctx slot goes after the model without a context segment",
			"segments = path model tokens",
			[]hook.Result{hookResult("a", hook.SlotCtx, "A")},
			"/srv/app · Opus · A · ↓ 1.5k · ↑ 2.0k",
		},
		{
			"ctx slot with no model goes after the last identity segment",
			"segments = tokens path",
			[]hook.Result{hookResult("a", hook.SlotCtx, "A")},
			"↓ 1.5k · ↑ 2.0k · /srv/app · A",
		},
		{
			"ctx slot alone is the whole line",
			"segments = tokens",
			[]hook.Result{hookResult("a", hook.SlotCtx, "A")},
			"A · ↓ 1.5k · ↑ 2.0k",
		},
		{
			"usage slot goes after tokens and before the rates",
			"segments = model tokens rate5h",
			[]hook.Result{hookResult("a", hook.SlotUsage, "U")},
			"Opus · ↓ 1.5k · ↑ 2.0k · U · 5h: 45%",
		},
		{
			"end slot goes last",
			"segments = model tokens",
			[]hook.Result{hookResult("a", hook.SlotEnd, "E")},
			"Opus · ↓ 1.5k · ↑ 2.0k · E",
		},
		{
			"hooks in one slot keep filename order",
			"segments = model",
			[]hook.Result{hookResult("a", hook.SlotEnd, "E1"), hookResult("b", hook.SlotEnd, "E2")},
			"Opus · E1 · E2",
		},
		{
			"a hook named in the segments list renders there",
			"segments = backup path model",
			[]hook.Result{hookResult("backup", hook.SlotEnd, "B")},
			"B · /srv/app · Opus",
		},
		{
			"a hook replaces the built-in of the same name",
			"segments = path model tokens",
			[]hook.Result{hookResult("model", hook.SlotCtx, "custom")},
			"/srv/app · custom · ↓ 1.5k · ↑ 2.0k",
		},
		{
			"a hook with nothing to show replaces the built-in with nothing",
			"segments = path model",
			[]hook.Result{hookResult("model", hook.SlotCtx, "")},
			"/srv/app",
		},
		{
			"a hook named like a built-in that is not listed goes to its slot",
			"segments = path",
			[]hook.Result{hookResult("model", hook.SlotEnd, "custom")},
			"/srv/app · custom",
		},
		{
			"empty hooks are dropped",
			"segments = path",
			[]hook.Result{hookResult("a", hook.SlotCtx, ""), hookResult("b", hook.SlotEnd, "")},
			"/srv/app",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, newLine(standardStubs(), tt.cfg, th, 999, tt.hooks).Render())
		})
	}
}

func TestRenderWrapsAroundHooks(t *testing.T) {
	t.Parallel()

	hooks := []hook.Result{hookResult("a", hook.SlotCtx, "ctx-hook"), hookResult("b", hook.SlotUsage, "usage-hook"), hookResult("c", hook.SlotEnd, "end-hook")}
	line := newLine(standardStubs(), "segments = path model tokens", plainClassic(), 40, hooks)
	assert.Equal(t, "/srv/app · Opus · ctx-hook\n↓ 1.5k · ↑ 2.0k\nusage-hook · end-hook", line.Render())
}

func TestRenderPaintsTheContinuationAnchor(t *testing.T) {
	t.Parallel()

	plugins := []*stub{
		{"path", layout.Identity, "/srv/app"},
		{"model", layout.Identity, "Opus 4.7 Extended Thinking"},
		{"ctx", layout.Identity, "ctx: ░░░░░░░░░░ 1%"},
	}
	line := newLine(plugins, "segments = path model ctx", theme.Load("classic", "", ""), 50, nil)
	assert.Contains(t, line.Render(), "\n\033[2m↳\033[0m          ctx: ")
}

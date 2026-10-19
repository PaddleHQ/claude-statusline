package render_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
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

func newLine(stubs []*stub, cfg string, th theme.Theme, columns int) *render.Line {
	c := config.New(cfg)
	return &render.Line{
		Names:     c.Strings("segments"),
		Lookup:    lookupIn(stubs...),
		Columns:   columns,
		Separator: " " + th.Paint(th.Muted, c.String("separator")) + " ",
		Anchor:    th.Paint(th.Muted, "↳"),
	}
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
		{"unknown names are skipped", "segments: [bogus, model, \"Bad-Name\", \"../x\"]", th, "Opus"},
		{"repeated segment is joined with the painted separator", "segments: [model, model]", th, "Opus \033[2m·\033[0m Opus"},
		{"custom separator is painted muted", "segments: [model, model]\nseparator: \"|\"", th, "Opus \033[2m|\033[0m Opus"},
		{"no color leaves the separator bare", "segments: [model, model]", th.WithoutColor(), "Opus · Opus"},
		{"only unknown segments render nothing", "segments: [bogus]", th, ""},
		{"empty segments falls back to the default", "segments =", th, "/srv/app \033[2m·\033[0m Opus \033[2m·\033[0m ctx: 42% \033[2m·\033[0m ↓ 1.5k · ↑ 2.0k \033[2m·\033[0m 5h: 45%"},
		{"separator injection stays data", "segments: [model, model]\nseparator: \"$(touch x)\"", th, "Opus \033[2m$(touch x)\033[0m Opus"},
		{"segments with nothing to show are dropped", "segments: [model, empty, model]", th, "Opus \033[2m·\033[0m Opus"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, newLine(standardStubs(), tt.cfg, tt.th, 0).Render())
		})
	}
}

func TestRenderPaintsTheContinuationAnchor(t *testing.T) {
	t.Parallel()

	plugins := []*stub{
		{"path", layout.Identity, "/srv/app"},
		{"model", layout.Identity, "Opus 4.7 Extended Thinking"},
		{"ctx", layout.Identity, "ctx: ░░░░░░░░░░ 1%"},
	}
	line := newLine(plugins, "segments: [path, model, ctx]", theme.Load("classic", "", ""), 50)
	assert.Contains(t, line.Render(), "\n\033[2m↳\033[0m          ctx: ")
}

package segment_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/hook"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/segment"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
)

func hookResult(name string, slot hook.Slot, text string) hook.Result {
	return hook.Result{Hook: hook.Hook{File: name + ".sh", Name: name, Slot: slot}, Text: text}
}

func TestRenderPlacesHooks(t *testing.T) {
	t.Parallel()

	const input = `{"model":{"display_name":"Opus"},"cwd":"/srv/app","context_window":{"used_percentage":42,"total_input_tokens":1500,"total_output_tokens":2000},` +
		`"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":1760009000}}}`
	th := theme.Load("classic", "", "").WithoutColor()

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
			"/srv/app · Opus · ctx:  42% · A · ↓ 1.5k · ↑ 2.0k",
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
			"Opus · ↓ 1.5k · ↑ 2.0k · U",
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
			ctx := &segment.Context{Payload: payload.Parse([]byte(input)), Config: config.New(tt.cfg), Theme: th, Columns: 999, Hooks: tt.hooks}
			assert.Equal(t, tt.want, stripBars(segment.Render(ctx)))
		})
	}
}

func TestRenderWrapsAroundHooks(t *testing.T) {
	t.Parallel()

	const input = `{"model":{"display_name":"Opus"},"cwd":"/srv/app","context_window":{"total_input_tokens":1500,"total_output_tokens":2000}}`
	hooks := []hook.Result{hookResult("a", hook.SlotCtx, "ctx-hook"), hookResult("b", hook.SlotUsage, "usage-hook"), hookResult("c", hook.SlotEnd, "end-hook")}
	ctx := &segment.Context{
		Payload: payload.Parse([]byte(input)),
		Config:  config.New("segments = path model tokens"),
		Theme:   theme.Load("classic", "", "").WithoutColor(),
		Columns: 40,
		Hooks:   hooks,
	}
	assert.Equal(t, "/srv/app · Opus · ctx-hook\n↓ 1.5k · ↑ 2.0k\nusage-hook · end-hook", segment.Render(ctx))
}

func TestRenderPaintsTheContinuationAnchor(t *testing.T) {
	t.Parallel()

	ctx := &segment.Context{
		Payload: payload.Parse([]byte(`{"model":{"display_name":"Opus 4.7 Extended Thinking"},"cwd":"/srv/app","context_window":{"used_percentage":1}}`)),
		Config:  config.New("segments = path model ctx"),
		Theme:   theme.Load("classic", "", ""),
		Columns: 50,
	}
	assert.Contains(t, segment.Render(ctx), "\n\033[2m↳\033[0m          ctx: ")
}

func stripBars(s string) string {
	out := []rune{}
	for _, r := range s {
		switch r {
		case '█', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '░', '┃':
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

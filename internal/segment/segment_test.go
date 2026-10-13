package segment_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/segment"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
)

func testPayload() payload.Payload {
	return payload.Parse([]byte(`{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"}}`))
}

func TestRegistryNamesMatch(t *testing.T) {
	t.Parallel()

	for name, s := range segment.Registry {
		assert.Equal(t, name, s.Name(), "segment registered as %q", name)
	}
}

func TestRender(t *testing.T) {
	t.Parallel()

	th := theme.Lookup("default")
	tests := []struct {
		name string
		cfg  string
		th   theme.Theme
		want string
	}{
		{"default segments", "", th, "Opus 4.7 [high]"},
		{"unknown names are skipped", "segments = bogus model Bad-Name ../x", th, "Opus 4.7 [high]"},
		{"repeated segment is joined with the painted separator", "segments = model model", th, "Opus 4.7 [high] \033[2m·\033[0m Opus 4.7 [high]"},
		{"custom separator is painted muted", "segments = model model\nseparator = |", th, "Opus 4.7 [high] \033[2m|\033[0m Opus 4.7 [high]"},
		{"no colour leaves the separator bare", "segments = model model", th.WithoutColor(), "Opus 4.7 [high] · Opus 4.7 [high]"},
		{"only unknown segments render nothing", "segments = bogus", th, ""},
		{"empty segments falls back to the default", "segments =", th, "Opus 4.7 [high]"},
		{"separator injection stays data", "segments = model model\nseparator = $(touch x)", th, "Opus 4.7 [high] \033[2m$(touch x)\033[0m Opus 4.7 [high]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := &segment.Context{Payload: testPayload(), Config: config.New(tt.cfg), Theme: tt.th}
			assert.Equal(t, tt.want, segment.Render(ctx))
		})
	}
}

func TestRenderDropsEmptySegments(t *testing.T) {
	t.Parallel()

	ctx := &segment.Context{Config: config.New("segments = model model"), Theme: theme.Lookup("default")}
	assert.Empty(t, segment.Render(ctx), "Render() with an empty payload")
}

func TestModelRender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		json  string
		cfg   string
		theme theme.Theme
		want  string
	}{
		{"name and effort", `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"}}`, "", theme.Lookup("default"), "Opus 4.7 [high]"},
		{"no effort", `{"model":{"display_name":"Sonnet 5.5"}}`, "", theme.Lookup("default"), "Sonnet 5.5"},
		{"effort hidden", `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"}}`, "model.effort = false", theme.Lookup("default"), "Opus 4.7"},
		{"invalid effort setting uses the default", `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"}}`, "model.effort = maybe", theme.Lookup("default"), "Opus 4.7 [high]"},
		{"no model", `{"effort":{"level":"high"}}`, "", theme.Lookup("default"), ""},
		{"painted with the text colour", `{"model":{"display_name":"Opus"}}`, "", theme.Theme{Text: "<t>", Reset: "</t>"}, "<t>Opus</t>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := &segment.Context{Payload: payload.Parse([]byte(tt.json)), Config: config.New(tt.cfg), Theme: tt.theme}
			model, ok := segment.Lookup("model")
			require.True(t, ok)
			assert.Equal(t, tt.want, model.Render(ctx))
		})
	}
}

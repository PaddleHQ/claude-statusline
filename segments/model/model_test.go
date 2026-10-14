package model_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/internal/transcript"
	"github.com/PaddleHQ/claude-statusline/segments/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func render(json, cfg string, th theme.Theme, info transcript.Info) string {
	providers := plugin.NewProviders(&plugin.Sources{
		Config:     config.New(cfg, model.Plugin().Defaults),
		Theme:      th,
		Transcript: func() transcript.Info { return info },
	})
	seg := model.New(providers.For("model"))
	return seg.Render(&plugin.Session{Payload: payload.Parse([]byte(json))})
}

func plain() theme.Theme {
	classic := theme.Load("classic", "", "")
	return classic.WithoutColor()
}

func TestRender(t *testing.T) {
	t.Parallel()

	classic := theme.Load("classic", "", "")
	tests := []struct {
		name  string
		json  string
		cfg   string
		theme theme.Theme
		want  string
	}{
		{"name and effort", `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"}}`, "", classic, "Opus 4.7 [high]"},
		{"no effort", `{"model":{"display_name":"Sonnet 5.5"}}`, "", classic, "Sonnet 5.5"},
		{"effort hidden", `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"}}`, "model.effort = false", classic, "Opus 4.7"},
		{"invalid effort setting uses the default", `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"}}`, "model.effort = maybe", classic, "Opus 4.7 [high]"},
		{"no model", `{"effort":{"level":"high"}}`, "", classic, ""},
		{"painted with the text color", `{"model":{"display_name":"Opus"}}`, "", theme.Theme{Text: "<t>", Reset: "</t>"}, "<t>Opus</t>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, render(tt.json, tt.cfg, tt.theme, transcript.Info{}))
		})
	}
}

func TestSkillOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		id     string
		actual string
		want   string
	}{
		{"same family", "claude-opus-4-7", "claude-opus-4-7-20260101", "Opus 4.7 [high]"},
		{"different family", "claude-opus-4-7", "claude-haiku-4-5", "Haiku (skill)"},
		{"every family has a label", "claude-opus-4-7", "claude-sonnet-5", "Sonnet (skill)"},
		{"fable", "claude-opus-4-7", "fable-5", "Fable (skill)"},
		{"mythos", "claude-opus-4-7", "mythos-1", "Mythos (skill)"},
		{"unknown actual family is ignored", "claude-opus-4-7", "something-else", "Opus 4.7 [high]"},
		{"no actual model", "claude-opus-4-7", "", "Opus 4.7 [high]"},
		{"unknown session family differs from a known one", "custom-model", "claude-haiku-4-5", "Haiku (skill)"},
		{"unknown on both sides", "custom-model", "other", "Opus 4.7 [high]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			json := `{"transcript_path":"/t.jsonl","model":{"id":"` + tt.id + `","display_name":"Opus 4.7"},"effort":{"level":"high"}}`
			got := render(json, "", plain(), transcript.Info{ActualModel: tt.actual})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIgnoresTheTranscriptWithoutAPath(t *testing.T) {
	t.Parallel()

	json := `{"model":{"id":"claude-opus-4-7","display_name":"Opus 4.7"}}`
	got := render(json, "", plain(), transcript.Info{ActualModel: "claude-haiku-4-5"})
	assert.Equal(t, "Opus 4.7", got)
}

func TestIsAnIdentitySegment(t *testing.T) {
	t.Parallel()

	entries := model.Plugin().Segments
	require.Len(t, entries, 1)
	seg := entries[0].New(plugin.NewProviders(&plugin.Sources{Config: config.New("")}))
	assert.Equal(t, entries[0].Name, seg.Name())
	assert.Equal(t, layout.Identity, seg.Group())
}

func TestHelp(t *testing.T) {
	t.Parallel()

	help := func(cfg string) string {
		providers := plugin.NewProviders(&plugin.Sources{Config: config.New(cfg, model.Plugin().Defaults)})
		return model.New(providers.For("model")).Help()
	}
	assert.Contains(t, help(""), "effort")
	assert.NotContains(t, help("model.effort = false"), "effort")
	assert.Contains(t, help("model.effort = false"), "model")
}

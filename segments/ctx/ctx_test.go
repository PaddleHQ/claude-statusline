package ctx_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments/ctx"

	"github.com/stretchr/testify/assert"
)

func renderOne(json, cfg string, th theme.Theme) string {
	providers := plugin.NewProviders(&plugin.Sources{
		Config:  config.New(cfg, ctx.Plugin().Defaults),
		Theme:   th,
		Columns: 999,
	})
	return ctx.New(providers.For("ctx")).Render(&plugin.Session{Payload: payload.Parse([]byte(json))})
}

func TestCtxRender(t *testing.T) {
	t.Parallel()

	markers := theme.Theme{Good: "<g>", Warn: "<w>", Bad: "<b>", Muted: "<d>", Reset: "</>", Full: "F", Partials: [7]string{"1", "2", "3", "4", "5", "6", "7"}, Empty: "E"}
	tests := []struct {
		name string
		json string
		cfg  string
		want string
	}{
		{"good below warn", `{"context_window":{"used_percentage":49}}`, "ctx.width = 4", "ctx: <g>FF</><d>EE</> 49%"},
		{"warn at the threshold", `{"context_window":{"used_percentage":50}}`, "ctx.width = 4", "ctx: <w>FF</><d>EE</> 50%"},
		{"bad at the threshold", `{"context_window":{"used_percentage":90}}`, "ctx.width = 4", "ctx: <b>FFF5</> 90%"},
		{"custom thresholds", `{"context_window":{"used_percentage":20}}`, "ctx.width = 4\nctx.warn = 10\nctx.bad = 20", "ctx: <b>6</><d>EEE</> 20%"},
		{"percentage rounds to even", `{"context_window":{"used_percentage":42.5}}`, "ctx.width = 4", "ctx: <g>F5</><d>EE</> 42%"},
		{"zero is shown", `{"context_window":{"used_percentage":0}}`, "ctx.width = 4", "ctx: <d>EEEE</> 0%"},
		{"absent hides the segment", `{"context_window":{}}`, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, renderOne(tt.json, tt.cfg, markers))
		})
	}
}

func TestHelp(t *testing.T) {
	t.Parallel()

	providers := plugin.NewProviders(&plugin.Sources{Config: config.New("ctx.warn = 40\nctx.bad = 80", ctx.Plugin().Defaults)})
	help := ctx.New(providers.For("ctx")).Help()
	assert.Contains(t, help, "40%")
	assert.Contains(t, help, "80%")
}

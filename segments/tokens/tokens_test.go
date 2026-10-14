package tokens_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments/tokens"

	"github.com/stretchr/testify/assert"
)

func renderOne(json, cfg string, th theme.Theme) string {
	providers := plugin.NewProviders(&plugin.Sources{Config: config.New(cfg), Theme: th})
	return tokens.New(providers.For("tokens")).Render(&plugin.Session{Payload: payload.Parse([]byte(json))})
}

func TestFormatTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0k"},
		{1049, "1.0k"},
		{1500, "1.5k"},
		{999999, "999.9k"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{12345678, "12.3M"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tokens.FormatTokens(tt.n))
		})
	}
}

func TestTokensRender(t *testing.T) {
	t.Parallel()

	painted := theme.Theme{Text: "<t>", Muted: "<m>", Reset: "</>"}
	tests := []struct {
		name string
		json string
		cfg  string
		th   theme.Theme
		want string
	}{
		{"both", `{"context_window":{"total_input_tokens":1500,"total_output_tokens":250}}`, "", painted, "<t>↓ 1.5k</> <m>·</> <t>↑ 250</>"},
		{"custom separator", `{"context_window":{"total_input_tokens":1,"total_output_tokens":2}}`, "separator = |", painted, "<t>↓ 1</> <m>|</> <t>↑ 2</>"},
		{"input only", `{"context_window":{"total_input_tokens":1500}}`, "", painted, "<t>↓ 1.5k</>"},
		{"output only", `{"context_window":{"total_output_tokens":7}}`, "", painted, "<t>↑ 7</>"},
		{"neither hides the segment", `{"context_window":{"used_percentage":5}}`, "", painted, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, renderOne(tt.json, tt.cfg, tt.th))
		})
	}
}

func TestHelp(t *testing.T) {
	t.Parallel()

	providers := plugin.NewProviders(&plugin.Sources{Config: config.New("")})
	assert.NotEmpty(t, tokens.New(providers.For("tokens")).Help())
}

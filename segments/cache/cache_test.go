package cache_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments/cache"

	"github.com/stretchr/testify/assert"
)

func renderOne(json, cfg string, th theme.Theme) string {
	providers := plugin.NewProviders(&plugin.Sources{Config: config.New(cfg, cache.Plugin().Defaults), Theme: th})
	return cache.New(providers.For("cache")).Render(&plugin.Session{Payload: payload.Parse([]byte(json))})
}

func TestCacheRender(t *testing.T) {
	t.Parallel()

	painted := theme.Theme{Text: "<t>", Bad: "<b>", Reset: "</>"}
	usage := func(in, create, read string) string {
		return `{"context_window":{"current_usage":{"input_tokens":` + in + `,"cache_creation_input_tokens":` + create + `,"cache_read_input_tokens":` + read + `}}}`
	}
	tests := []struct {
		name string
		json string
		cfg  string
		want string
	}{
		{"healthy", usage("10", "10", "80"), "", "<t>⧉ 80%</>"},
		{"at the threshold is not red", usage("95", "0", "5"), "", "<t>⧉ 5%</>"},
		{"below the threshold is red", usage("96", "0", "4"), "", "<b>⧉ 4%</>"},
		{"integer division truncates", usage("1", "0", "2"), "cache.warn = 0", "<t>⧉ 66%</>"},
		{"custom threshold", usage("10", "10", "80"), "cache.warn = 90", "<b>⧉ 80%</>"},
		{"all reads is 100", usage("0", "0", "10"), "", "<t>⧉ 100%</>"},
		{"zero denominator hides the segment", usage("0", "0", "0"), "", ""},
		{"no usage hides the segment", `{"context_window":{}}`, "", ""},
		{"missing counts are zero", `{"context_window":{"current_usage":{"cache_read_input_tokens":5}}}`, "", "<t>⧉ 100%</>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, renderOne(tt.json, tt.cfg, painted))
		})
	}
}

func TestHelp(t *testing.T) {
	t.Parallel()

	providers := plugin.NewProviders(&plugin.Sources{Config: config.New("cache.warn = 12", cache.Plugin().Defaults)})
	assert.Contains(t, cache.New(providers.For("cache")).Help(), "12%")
}

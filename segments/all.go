// Package segments is the one list of plugins the status line can show.
package segments

import (
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/segments/cache"
	"github.com/PaddleHQ/claude-statusline/segments/ctx"
	"github.com/PaddleHQ/claude-statusline/segments/diagnostics"
	"github.com/PaddleHQ/claude-statusline/segments/model"
	"github.com/PaddleHQ/claude-statusline/segments/path"
	"github.com/PaddleHQ/claude-statusline/segments/rate"
	"github.com/PaddleHQ/claude-statusline/segments/tokens"
)

// All returns every plugin. To add one, add a line here.
func All() []*plugin.Plugin {
	return []*plugin.Plugin{
		path.Plugin(),
		model.Plugin(),
		ctx.Plugin(),
		tokens.Plugin(),
		cache.Plugin(),
		rate.Plugin(),
		diagnostics.Plugin(),
	}
}

// Defaults returns the text of every plugin's defaults.conf, for layering under the user's config.
func Defaults() []string {
	var texts []string
	for _, p := range All() {
		if p.Defaults != "" {
			texts = append(texts, p.Defaults)
		}
	}
	return texts
}

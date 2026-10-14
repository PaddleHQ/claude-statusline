// Package segments is the one list of plugins the status line can show.
package segments

import (
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/internal/segment"
	"github.com/PaddleHQ/claude-statusline/segments/model"
)

// All returns every plugin. To add one, add a line here.
func All() []*plugin.Plugin {
	return []*plugin.Plugin{
		model.Plugin(),
		segment.Plugin(),
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

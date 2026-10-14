package render

import (
	"github.com/PaddleHQ/claude-statusline/internal/layout"
)

// items renders the configured segments in order, leaving out the unknown names and the segments with nothing to show.
func (l *Line) items() []layout.Item {
	var out []layout.Item
	for _, name := range l.Names {
		builtin, group, ok := l.Lookup(name)
		if !ok {
			continue
		}
		if text := builtin.Render(); text != "" {
			out = append(out, layout.Item{Group: group, Text: text})
		}
	}
	return out
}

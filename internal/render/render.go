// Package render lays out the configured segments and the user's hooks into the printed line.
package render

import (
	"github.com/PaddleHQ/claude-statusline/internal/hook"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
)

const (
	keySegments  = "segments"
	keySeparator = "separator"
	nameCtx      = "ctx"
	nameModel    = "model"
)

// Keys lists the config keys the renderer itself reads.
func Keys() []string { return []string{keySegments, keySeparator} }

// Line is one render: the plugins that may be listed in segments, what they draw from, and the hooks to place among them.
type Line struct {
	Plugins   []*plugin.Plugin
	Providers *plugin.Providers
	Session   *plugin.Session
	// Hooks holds the output of the user's hook scripts, in filename order.
	Hooks []hook.Result
}

// Render lays out the configured segments and the hooks, wrapping onto further rows when they do not fit.
func (l *Line) Render() string {
	th := l.Providers.Theme
	opts := &layout.Options{
		Columns:   l.Providers.Columns,
		Separator: l.Providers.Separator(),
		Anchor:    th.Paint(th.Muted, "↳"),
	}
	return opts.Render(l.items())
}

// lookup builds the segment with the given name, if some plugin provides it.
func (l *Line) lookup(name string) (plugin.Segment, bool) {
	for _, pl := range l.Plugins {
		for _, e := range pl.Segments {
			if e.Name == name {
				return e.New(l.Providers.For(name)), true
			}
		}
	}
	return nil, false
}

// Package render lays out the configured segments and the user's hooks into the printed line.
package render

import (
	"github.com/PaddleHQ/claude-statusline/internal/hook"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
)

const (
	keySeparator = "separator"
	nameCtx      = "ctx"
	nameModel    = "model"
)

// Keys lists the config keys the renderer itself reads.
func Keys() []string { return []string{keySeparator} }

// Segment is a built-in segment, ready to draw.
type Segment interface {
	// Render returns the segment's text, or "" to hide it.
	Render() string
}

// Line is one render: the segments to show, and how to find, join and wrap them.
type Line struct {
	// Names is the segments list from the config, in order, less segments_excluded; a hook whose name is left out
	// goes to its slot instead.
	Names []string
	// Lookup builds the built-in segment with a name, with the layout group it belongs to.
	Lookup  func(name string) (Segment, layout.Group, bool)
	Columns int
	// Separator is the painted glyph drawn between segments, with a space either side.
	Separator string
	// Anchor is drawn at the start of an indented row.
	Anchor string
	// Hooks holds the output of the user's hook scripts, in filename order.
	Hooks []hook.Result
}

// Render lays out the configured segments and the hooks, wrapping onto further rows when they do not fit.
func (l *Line) Render() string {
	opts := &layout.Options{Columns: l.Columns, Separator: l.Separator, Anchor: l.Anchor}
	return opts.Render(l.items())
}

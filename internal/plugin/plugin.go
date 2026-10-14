// Package plugin defines what a status line segment is and what it may use to draw itself.
package plugin

import (
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
)

// Segment renders one part of the line.
type Segment interface {
	Name() string
	// Group says which part of the line the segment belongs to when it wraps.
	Group() layout.Group
	// Keys lists every config key Render reads, relative to the segment's section, so a test can check them against defaults.conf.
	Keys() []string
	// Help says in a sentence or two what the segment shows and what its colors mean,
	// using the user's current settings for any threshold or label.
	Help() string
	// Render returns the segment's text, or "" to hide it.
	Render(*Session) string
}

// Session is what one render is about: the payload Claude Code sent.
type Session struct {
	Payload payload.Payload
}

// Entry names a segment and builds it.
type Entry struct {
	Name string
	// New is called with the providers scoped to Name. It must be cheap: the expensive providers are only asked for in Render.
	New func(*Providers) Segment
}

// Plugin is what a package under segments/ provides: its segments, and the text of its defaults.conf.
type Plugin struct {
	Defaults string
	Segments []Entry
}

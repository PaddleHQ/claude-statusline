// Package segment defines the status line's building blocks and renders them in order.
package segment

import (
	"strings"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
)

// Segment renders one part of the line. Keys lists every config key Render reads,
// so a test can check them against defaults.conf.
type Segment interface {
	Name() string
	Keys() []string
	Render(*Context) string
}

// Context is everything a segment may read.
type Context struct {
	Payload payload.Payload
	Config  *config.Config
	Theme   theme.Theme
	Now     time.Time
	Columns int
}

const (
	keySegments  = "segments"
	keySeparator = "separator"
)

var registry = map[string]Segment{
	"model": model{},
}

// Lookup returns the registered segment with the given name.
func Lookup(name string) (Segment, bool) {
	s, ok := registry[name]
	return s, ok
}

// Keys lists every config key read by Render and by the registered segments.
func Keys() []string {
	keys := []string{keySegments, keySeparator}
	for _, s := range registry {
		keys = append(keys, s.Keys()...)
	}
	return keys
}

// Render joins the configured segments in order. Unknown names and empty renders are dropped.
func Render(ctx *Context) string {
	sep := " " + ctx.Theme.Paint(ctx.Theme.Muted, ctx.Config.String(keySeparator)) + " "
	var parts []string
	for _, name := range strings.Fields(ctx.Config.String(keySegments)) {
		s, ok := Lookup(name)
		if !ok {
			continue
		}
		if out := s.Render(ctx); out != "" {
			parts = append(parts, out)
		}
	}
	return strings.Join(parts, sep)
}

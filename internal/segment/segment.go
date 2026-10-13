// Package segment defines the status line's building blocks and renders them in order.
package segment

import (
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/hook"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
)

// Segment renders one part of the line. Keys lists every config key Render reads,
// so a test can check them against defaults.conf.
type Segment interface {
	Name() string
	// Group says which part of the line the segment belongs to when it wraps.
	Group() layout.Group
	Keys() []string
	Render(*Context) string
}

// Context is everything a segment may read.
type Context struct {
	Payload payload.Payload
	// Rates is the merged, still-valid rate-limit reading, not the raw payload.
	Rates   ratecache.Limits
	Config  *config.Config
	Theme   theme.Theme
	Now     time.Time
	Columns int
	Home    string
	// Hooks holds the output of the user's hook scripts, in filename order.
	Hooks []hook.Result
}

const (
	keySegments  = "segments"
	keySeparator = "separator"
)

var registry = map[string]Segment{
	"path":   path{},
	"model":  model{},
	"ctx":    ctxBar{},
	"tokens": tokens{},
	"cache":  cache{},
	"rate5h": rate{name: "rate5h", seconds: fiveHourSeconds, reading: func(c *Context) *ratecache.Reading { return c.Rates.FiveHour }},
	"rate7d": rate{name: "rate7d", seconds: sevenDaySeconds, reading: func(c *Context) *ratecache.Reading { return c.Rates.SevenDay }},
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

// Render lays out the configured segments and the user's hooks, wrapping onto further rows when they do not fit.
func Render(ctx *Context) string {
	return layout.Render(items(ctx), layout.Options{
		Columns:   ctx.Columns,
		Separator: separator(ctx),
		Anchor:    ctx.Theme.Paint(ctx.Theme.Muted, "↳"),
	})
}

func separator(ctx *Context) string {
	return " " + ctx.Theme.Paint(ctx.Theme.Muted, ctx.Config.String(keySeparator)) + " "
}

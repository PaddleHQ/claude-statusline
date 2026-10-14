// Package segment holds the segments not yet converted to plugins under segments/, and bridges them to the plugin interface.
package segment

import (
	"sort"
	"strings"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/internal/transcript"
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
	Config  *config.Section
	Theme   theme.Theme
	Now     time.Time
	Columns int
	Home    string
	// Transcript is what the session transcript revealed; zero when there is none.
	Transcript transcript.Info
}

const nameCtx = "ctx"

var registry = map[string]Segment{
	"path":   &path{},
	nameCtx:  &ctxBar{},
	"tokens": &tokens{},
	"cache":  &cache{},
	"rate5h": &rate{name: "rate5h", seconds: fiveHourSeconds, reading: func(c *Context) *ratecache.Reading { return c.Rates.FiveHour }},
	"rate7d": &rate{name: "rate7d", seconds: sevenDaySeconds, reading: func(c *Context) *ratecache.Reading { return c.Rates.SevenDay }},
}

func init() {
	for i := range diagnostics {
		d := &diagnostics[i]
		registry[d.name] = d
	}
}

// Lookup returns the registered segment with the given name.
func Lookup(name string) (Segment, bool) {
	s, ok := registry[name]
	return s, ok
}

// Plugin exposes the segments in this package as one plugin. Their defaults are still in the global defaults.conf.
func Plugin() *plugin.Plugin {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	entries := make([]plugin.Entry, 0, len(names))
	for _, name := range names {
		entries = append(entries, plugin.Entry{Name: name, New: func(p *plugin.Providers) plugin.Segment {
			return &bridge{inner: registry[name], p: p}
		}})
	}
	return &plugin.Plugin{Segments: entries}
}

type bridge struct {
	inner Segment
	p     *plugin.Providers
}

func (b *bridge) Name() string { return b.inner.Name() }

func (b *bridge) Group() layout.Group { return b.inner.Group() }

func (b *bridge) Help() string { return "" }

func (b *bridge) Keys() []string {
	keys := b.inner.Keys()
	for i, k := range keys {
		keys[i] = strings.TrimPrefix(k, b.inner.Name()+".")
	}
	return keys
}

func (b *bridge) Render(s *plugin.Session) string {
	p := b.p.For("")
	return b.inner.Render(&Context{
		Payload:    s.Payload,
		Rates:      p.RateLimits(),
		Config:     p.Config,
		Theme:      p.Theme,
		Now:        p.Now(),
		Columns:    p.Columns,
		Home:       p.Home,
		Transcript: p.Transcript(),
	})
}

// barWidth reads the three widths configured for a bar and picks the one for the terminal.
func barWidth(ctx *Context, name string) int {
	cfg := ctx.Config
	return bar.WidthFor(ctx.Columns, cfg.Int(name+".width"), cfg.Int(name+".width_medium"), cfg.Int(name+".width_narrow"))
}

func separator(ctx *Context) string {
	return " " + ctx.Theme.Paint(ctx.Theme.Muted, ctx.Config.String("separator")) + " "
}

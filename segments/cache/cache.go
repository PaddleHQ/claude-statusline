// Package cache provides the cache segment: how much of the latest request's input came from the prompt cache.
package cache

import (
	_ "embed"
	"fmt"
	"strconv"

	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
)

const (
	name         = "cache"
	percentScale = 100
)

//go:embed defaults.conf
var defaults string

// Plugin is the cache plugin.
func Plugin() *plugin.Plugin {
	return &plugin.Plugin{Defaults: defaults, Segments: []plugin.Entry{{Name: name, New: New}}}
}

// New builds the cache segment.
func New(p *plugin.Providers) plugin.Segment { return &cache{p: p} }

type cache struct{ p *plugin.Providers }

func (c *cache) Name() string { return name }

func (c *cache) Group() layout.Group { return layout.Usage }

func (c *cache) Help() string {
	return fmt.Sprintf("How much of the latest request's input was read from the prompt cache. Red below %d%%.", c.p.Config.Int("warn"))
}

func (c *cache) Keys() []string { return []string{"warn"} }

func (c *cache) Render(s *plugin.Session) string {
	th := c.p.Theme
	u := s.Payload.ContextWindow.Current
	total := u.Input + u.CacheCreation + u.CacheRead
	if total <= 0 {
		return ""
	}
	pct := u.CacheRead * percentScale / total
	color := th.Text
	if pct < int64(c.p.Config.Int("warn")) {
		color = th.Bad
	}
	return th.Paint(color, "⧉ "+strconv.FormatInt(pct, 10)+"%")
}

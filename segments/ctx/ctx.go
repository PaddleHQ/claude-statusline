// Package ctx provides the ctx segment: context-window usage as a bar.
package ctx

import (
	_ "embed"
	"fmt"
	"math"
	"strconv"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
)

const name = "ctx"

//go:embed defaults.conf
var defaults string

// Plugin is the ctx plugin.
func Plugin() *plugin.Plugin {
	return &plugin.Plugin{Defaults: defaults, Segments: []plugin.Entry{{Name: name, New: New}}}
}

// New builds the ctx segment.
func New(p *plugin.Providers) plugin.Segment { return &ctxBar{p: p} }

// ctxBar draws context-window usage as "ctx: <bar> <NN>%", colored by absolute usage.
type ctxBar struct{ p *plugin.Providers }

func (c *ctxBar) Name() string { return name }

func (c *ctxBar) Group() layout.Group { return layout.Identity }

func (c *ctxBar) Help() string {
	cfg := c.p.Config
	return fmt.Sprintf("How full the context window is. Amber from %d%%, red from %d%%.", cfg.Int("warn"), cfg.Int("bad"))
}

func (c *ctxBar) Keys() []string {
	return []string{"width", "width_medium", "width_narrow", "warn", "bad"}
}

func (c *ctxBar) Render(s *plugin.Session) string {
	used := s.Payload.ContextWindow.UsedPercentage
	if used == nil {
		return ""
	}
	pct := int(math.RoundToEven(*used))
	params := &bar.Params{
		Usage:    pct,
		Width:    c.p.BarWidth(),
		Warn:     c.p.Config.Int("warn"),
		Bad:      c.p.Config.Int("bad"),
		NoTarget: true,
	}
	return Draw(c.p.Theme, params, params.PercentColor(c.p.Theme))
}

// Draw renders "ctx: <bar> <NN>%" for a bar without a target; the subagent rows draw the same bar.
func Draw(th theme.Theme, params *bar.Params, percentColor string) string {
	return th.Paint(th.Text, "ctx: ") + params.Render(th) + " " + th.Paint(percentColor, strconv.Itoa(params.Usage)+"%")
}

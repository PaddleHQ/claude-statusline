// Package ctx provides the ctx segment: context-window usage as a bar.
package ctx

import (
	_ "embed"
	"fmt"
	"math"
)

// Defaults is the text of the segment's defaults.yaml, for layering under the user's config.
//
//go:embed defaults.yaml
var Defaults string

const name = "ctx"

//go:generate go run github.com/vektra/mockery/v2 --name=Session

// Session is what the segment needs to know about the session.
type Session interface {
	// UsedPercentage is how full the context window is; false means the payload did not report it.
	UsedPercentage() (float64, bool)
}

//go:generate go run github.com/vektra/mockery/v2 --name=Config

// Config reads the segment's settings.
type Config interface {
	Int(key string) int
}

//go:generate go run github.com/vektra/mockery/v2 --name=Painter

// Painter colors text.
type Painter interface {
	Text(s string) string
}

//go:generate go run github.com/vektra/mockery/v2 --name=Meter

// Meter draws a bar with a percentage after it, and has no pace marker.
type Meter interface {
	// Usage draws the bar for a usage percentage. Warn and bad are usage percentages at which the fill changes color.
	Usage(usage, warn, bad int) string
}

// Deps is everything the segment is built from.
type Deps struct {
	Session Session
	Config  Config
	Paint   Painter
	Meter   Meter
}

// Context draws context-window usage as "ctx: <bar> <NN>%", colored by absolute usage.
type Context struct{ d Deps }

// New builds the ctx segment.
func New(d Deps) *Context { return &Context{d: d} }

// Name is the segment's name in the segments list.
func (c *Context) Name() string { return name }

// Help says what the segment shows, for the help command.
func (c *Context) Help() string {
	return fmt.Sprintf("How full the context window is. Amber from %d%%, red from %d%%.", c.d.Config.Int("warn"), c.d.Config.Int("bad"))
}

// Keys lists the config keys the segment reads, relative to its section; the widths are read by the Meter.
func (c *Context) Keys() []string {
	return []string{"width", "width_medium", "width_narrow", "warn", "bad"}
}

// Render returns the segment's text, or "" to hide it.
func (c *Context) Render() string {
	used, ok := c.d.Session.UsedPercentage()
	if !ok {
		return ""
	}
	pct := int(math.RoundToEven(used))
	return c.d.Paint.Text("ctx: ") + c.d.Meter.Usage(pct, c.d.Config.Int("warn"), c.d.Config.Int("bad"))
}

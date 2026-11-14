// Package ctx provides the ctx segment: context-window usage as a bar.
package ctx

import (
	_ "embed"
	"fmt"
	"math"
	"slices"
)

// Defaults is the text of the segment's defaults.yaml, for layering under the user's config.
//
//go:embed defaults.yaml
var Defaults string

const (
	name         = "ctx"
	percentScale = 100
)

//go:generate go run github.com/vektra/mockery/v2 --name=Session

// Session is what the segment needs to know about the session.
type Session interface {
	// UsedPercentage is how full the context window is; false means the payload did not report it.
	UsedPercentage() (float64, bool)
	// WindowSize is the context window's size in tokens; false means the payload did not report it.
	WindowSize() (int64, bool)
	// LatestRequest is the token usage of the latest request.
	LatestRequest() (input, cacheCreation, cacheRead int64)
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
	Bad(s string) string
}

//go:generate go run github.com/vektra/mockery/v2 --name=Meter

// Meter draws a bar with a percentage after it, and has no pace marker.
type Meter interface {
	// Usage draws the bar for a usage percentage. Warn and bad are usage percentages at which the fill changes color.
	Usage(usage, warn, bad int) string
	// Limit draws the bar with a marker at the limit percentage, none when it is outside 0 to 99. Reached makes the fill bad.
	Limit(usage, limit int, reached bool, warn, bad int) string
}

//go:generate go run github.com/vektra/mockery/v2 --name=Icons

// Icons picks a label for the user's icon set.
type Icons interface {
	Pick(plain, nerd string) string
}

// Deps is everything the segment is built from.
type Deps struct {
	Session Session
	Config  Config
	Paint   Painter
	Meter   Meter
	Icons   Icons
}

// Context draws context-window usage as "ctx: <bar> <NN>%", colored by absolute usage, with a compact marker and warning.
type Context struct{ d Deps }

// New builds the ctx segment.
func New(d Deps) *Context { return &Context{d: d} }

// Name is the segment's name in the segments list.
func (c *Context) Name() string { return name }

// Help says what the segment shows, for the help command.
func (c *Context) Help() string {
	base := fmt.Sprintf("How full the context window is. Amber from %d%%, red from %d%%.", c.d.Config.Int("warn"), c.d.Config.Int("bad"))
	at, pct := c.d.Config.Int("compact_at"), c.d.Config.Int("compact_pct")
	var where string
	switch {
	case at > 0 && pct > 0:
		where = fmt.Sprintf("at %d tokens or %d%% of the window, whichever is lower", at, pct)
	case at > 0:
		where = fmt.Sprintf("at %d tokens", at)
	case pct > 0:
		where = fmt.Sprintf("at %d%% of the window", pct)
	default:
		return base
	}
	return fmt.Sprintf("%s A marker shows where to /compact, %s, and a red warning once you reach it.", base, where)
}

// Keys lists the config keys the segment reads, relative to its section; the widths are read by the Meter.
func (c *Context) Keys() []string {
	return []string{"width", "width_medium", "width_narrow", "warn", "bad", "compact_at", "compact_pct"}
}

// Render returns the segment's text, or "" to hide it.
func (c *Context) Render() string {
	used, ok := c.d.Session.UsedPercentage()
	if !ok {
		return ""
	}
	pct := int(math.RoundToEven(used))
	label := c.d.Paint.Text(c.d.Icons.Pick("ctx: ", "\U000F035B "))
	warn, bad := c.d.Config.Int("warn"), c.d.Config.Int("bad")

	tokens := c.tokens()
	window := c.window(tokens, used)
	threshold := c.threshold(window)
	if threshold <= 0 {
		return label + c.d.Meter.Usage(pct, warn, bad)
	}

	marker := -1
	if window > 0 {
		marker = int(math.Round(float64(threshold) * percentScale / float64(window)))
	}
	reached := tokens >= threshold
	out := label + c.d.Meter.Limit(pct, marker, reached, warn, bad)
	if reached {
		out += " " + c.d.Paint.Bad(c.d.Icons.Pick("\u26a0 /compact", "\uf071 /compact"))
	}
	return out
}

func (c *Context) tokens() int64 {
	input, creation, read := c.d.Session.LatestRequest()
	return input + creation + read
}

// window is the context window size in tokens, inferred from the usage percentage when the payload omits it; 0 means unknown.
func (c *Context) window(tokens int64, used float64) int64 {
	if size, ok := c.d.Session.WindowSize(); ok && size > 0 {
		return size
	}
	if tokens > 0 && used > 0 {
		return int64(math.Round(float64(tokens) * percentScale / used))
	}
	return 0
}

// threshold is the lowest enabled compact limit in tokens; 0 means none applies.
func (c *Context) threshold(window int64) int64 {
	var limits []int64
	if at := c.d.Config.Int("compact_at"); at > 0 {
		limits = append(limits, int64(at))
	}
	if pct := c.d.Config.Int("compact_pct"); pct > 0 && window > 0 {
		limits = append(limits, int64(math.Round(float64(window)*float64(pct)/percentScale)))
	}
	if len(limits) == 0 {
		return 0
	}
	return slices.Min(limits)
}

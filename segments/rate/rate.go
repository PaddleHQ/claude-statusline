// Package rate provides the rate5h and rate7d segments: usage of the 5 hour and 7 day windows as a bar with a pace marker.
package rate

import (
	_ "embed"
	"time"
)

// Defaults is the text of the segment's defaults.conf, for layering under the user's config.
//
//go:embed defaults.conf
var Defaults string

const (
	fiveHour        = "rate5h"
	sevenDay        = "rate7d"
	fiveHourSeconds = 5 * 60 * 60
	sevenDaySeconds = 7 * 24 * 60 * 60
	percentScale    = 100
)

//go:generate go run github.com/vektra/mockery/v2 --name=Limits

// Limits holds the readings of both windows: the usage percentage and the Unix time the window resets, or false when there is none.
type Limits interface {
	FiveHourReading() (used, resetsAt int64, ok bool)
	SevenDayReading() (used, resetsAt int64, ok bool)
}

//go:generate go run github.com/vektra/mockery/v2 --name=Config

// Config reads the segment's settings.
type Config interface {
	String(key string) string
	Int(key string) int
}

//go:generate go run github.com/vektra/mockery/v2 --name=Painter

// Painter colors text.
type Painter interface {
	Text(s string) string
}

//go:generate go run github.com/vektra/mockery/v2 --name=Meter

// Meter draws a bar with a pace marker and a percentage after it.
type Meter interface {
	// Pace draws the bar for a usage percentage against a target percentage. Warn and bad are deviations from the
	// target in percentage points, and the fill is always bad from hardCap.
	Pace(usage, target, warn, bad, hardCap int) string
}

// Deps is everything the segments are built from. Limits is a function so that the rate cache is only read when a bar is drawn.
type Deps struct {
	Config Config
	Paint  Painter
	Meter  Meter
	Now    time.Time
	Limits func() Limits
}

// Window draws one usage window as "<label>: <bar> <NN>%".
type Window struct {
	d       Deps
	name    string
	window  string
	seconds int64
	reading func(Limits) (used, resetsAt int64, ok bool)
}

// NewFiveHour builds the rate5h segment.
func NewFiveHour(d Deps) *Window {
	return &Window{d: d, name: fiveHour, window: "5 hour", seconds: fiveHourSeconds, reading: Limits.FiveHourReading}
}

// NewSevenDay builds the rate7d segment.
func NewSevenDay(d Deps) *Window {
	return &Window{d: d, name: sevenDay, window: "7 day", seconds: sevenDaySeconds, reading: Limits.SevenDayReading}
}

// Name is the segment's name in the segments list.
func (w *Window) Name() string { return w.name }

// Keys lists the config keys the segment reads, relative to its section; the widths are read by the Meter.
func (w *Window) Keys() []string {
	return []string{"label", "width", "width_medium", "width_narrow", "warn", "bad", "hard_cap"}
}

// Render returns the segment's text, or "" to hide it.
func (w *Window) Render() string {
	used, resetsAt, ok := w.reading(w.d.Limits())
	if !ok {
		return ""
	}
	cfg := w.d.Config
	target := elapsedPercent(w.d.Now.Unix(), resetsAt, w.seconds)
	bar := w.d.Meter.Pace(int(used), target, cfg.Int("warn"), cfg.Int("bad"), cfg.Int("hard_cap"))
	return w.d.Paint.Text(cfg.String("label")+": ") + bar
}

// elapsedPercent is how much of a window, ending at resetsAt, has passed by now, clamped to 0..100.
func elapsedPercent(now, resetsAt, windowSeconds int64) int {
	elapsed := min(max(now-(resetsAt-windowSeconds), 0), windowSeconds)
	return int(elapsed * percentScale / windowSeconds)
}

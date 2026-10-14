// Package rate provides the rate5h and rate7d segments: usage of the 5 hour and 7 day windows as a bar with a pace marker.
package rate

import (
	_ "embed"
	"fmt"
	"strconv"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
)

const (
	fiveHour        = "rate5h"
	sevenDay        = "rate7d"
	fiveHourSeconds = 5 * 60 * 60
	sevenDaySeconds = 7 * 24 * 60 * 60
)

//go:embed defaults.conf
var defaults string

// Plugin is the rate plugin, providing one segment per window.
func Plugin() *plugin.Plugin {
	return &plugin.Plugin{Defaults: defaults, Segments: []plugin.Entry{
		{Name: fiveHour, New: func(p *plugin.Providers) plugin.Segment {
			return &rate{p: p, name: fiveHour, window: "5 hour", seconds: fiveHourSeconds, reading: fiveHourReading}
		}},
		{Name: sevenDay, New: func(p *plugin.Providers) plugin.Segment {
			return &rate{p: p, name: sevenDay, window: "7 day", seconds: sevenDaySeconds, reading: sevenDayReading}
		}},
	}}
}

func fiveHourReading(l *ratecache.Limits) *ratecache.Reading { return l.FiveHour }

func sevenDayReading(l *ratecache.Limits) *ratecache.Reading { return l.SevenDay }

// rate draws one usage window as "<label>: <bar> <NN>%".
type rate struct {
	p       *plugin.Providers
	name    string
	window  string
	seconds int64
	reading func(*ratecache.Limits) *ratecache.Reading
}

func (r *rate) Name() string { return r.name }

func (r *rate) Group() layout.Group { return layout.Rates }

func (r *rate) Help() string {
	cfg := r.p.Config
	return fmt.Sprintf("Your %s usage, with a marker at an even pace. "+
		"Amber when more than %d points off it, red beyond %d points or from %d%% used.",
		r.window, cfg.Int("warn"), cfg.Int("bad"), cfg.Int("hard_cap"))
}

func (r *rate) Keys() []string {
	return []string{"label", "width", "width_medium", "width_narrow", "warn", "bad", "hard_cap"}
}

func (r *rate) Render(*plugin.Session) string {
	limits := r.p.RateLimits()
	reading := r.reading(&limits)
	if reading == nil {
		return ""
	}
	th := r.p.Theme
	cfg := r.p.Config
	params := &bar.Params{
		Usage:   int(reading.Used),
		Target:  bar.ElapsedPercent(r.p.Now().Unix(), reading.ResetsAt, r.seconds),
		Width:   r.p.BarWidth(),
		Warn:    cfg.Int("warn"),
		Bad:     cfg.Int("bad"),
		HardCap: cfg.Int("hard_cap"),
	}
	label := th.Paint(th.Text, cfg.String("label")+": ")
	return label + params.Render(th) + " " + th.Paint(params.PercentColor(th), strconv.FormatInt(reading.Used, 10)+"%")
}

package plugin

import (
	"sync"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/internal/transcript"
)

// Sources is where the providers get their data. Transcript and RateLimits are called at most once per render, and only if a segment asks.
type Sources struct {
	Config     *config.Config
	Theme      theme.Theme
	Now        time.Time
	Columns    int
	Home       string
	Transcript func() transcript.Info
	RateLimits func() ratecache.Limits
}

// Providers is the shared, read-only data any segment may use.
type Providers struct {
	Theme theme.Theme
	// Config is scoped to the segment: a segment named model reads "effort" to get model.effort.
	Config  *config.Section
	Columns int
	Home    string

	src        *Sources
	now        time.Time
	transcript func() transcript.Info
	rateLimits func() ratecache.Limits
}

// NewProviders returns providers with an unscoped Config, which is what the renderer reads "segments" and "separator" from.
func NewProviders(src *Sources) *Providers {
	return &Providers{
		Theme:      src.Theme,
		Config:     src.Config.Section(""),
		Columns:    src.Columns,
		Home:       src.Home,
		src:        src,
		now:        src.Now,
		transcript: sync.OnceValue(orZero(src.Transcript)),
		rateLimits: sync.OnceValue(orZero(src.RateLimits)),
	}
}

// For returns the same providers with Config scoped to the named segment. The lazy providers stay shared.
func (p *Providers) For(name string) *Providers {
	scoped := *p
	scoped.Config = p.src.Config.Section(name)
	return &scoped
}

// Now is the time of this render.
func (p *Providers) Now() time.Time { return p.now }

// Transcript is what the session transcript revealed; the transcript is read on the first call.
func (p *Providers) Transcript() transcript.Info { return p.transcript() }

// RateLimits is the merged, still-valid rate-limit reading; the shared cache is synced on the first call.
func (p *Providers) RateLimits() ratecache.Limits { return p.rateLimits() }

// Separator is the painted glyph drawn between segments, with a space either side.
func (p *Providers) Separator() string {
	return " " + p.Theme.Paint(p.Theme.Muted, p.src.Config.String("separator")) + " "
}

// BarWidth is the width of a bar for this terminal, from the segment's width, width_medium and width_narrow settings.
func (p *Providers) BarWidth() int { return bar.WidthOf(p.Columns, p.Config) }

func orZero[T any](f func() T) func() T {
	if f == nil {
		return func() T { var zero T; return zero }
	}
	return f
}

// Package segments is the composition root of the status line: the one list of plugins it can show, and the one place
// that connects them to the rest of the program. Plugins declare what they need as interfaces in their own packages,
// and the program's internal types satisfy those interfaces directly.
package segments

import (
	"sync"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/gitbranch"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/internal/transcript"
	"github.com/PaddleHQ/claude-statusline/segments/cache"
	"github.com/PaddleHQ/claude-statusline/segments/ctx"
	"github.com/PaddleHQ/claude-statusline/segments/diagnostics"
	"github.com/PaddleHQ/claude-statusline/segments/model"
	"github.com/PaddleHQ/claude-statusline/segments/path"
	"github.com/PaddleHQ/claude-statusline/segments/rate"
	"github.com/PaddleHQ/claude-statusline/segments/tokens"
)

// Segment is what every plugin's segment provides.
type Segment interface {
	Name() string
	// Help says in a sentence or two what the segment shows and what its colors mean,
	// using the user's current settings for any threshold or label.
	Help() string
	// Keys lists every config key Render reads, relative to the segment's section, so a test can check them against defaults.yaml.
	Keys() []string
	// Render returns the segment's text, or "" to hide it.
	Render() string
}

// Defaults returns the text of every plugin's defaults.yaml, for layering under the user's config.
func Defaults() []string {
	return []string{path.Defaults, model.Defaults, ctx.Defaults, cache.Defaults, rate.Defaults}
}

// Env is one render's world, from which the segments are built. Transcript and RateLimits are called at most once per render,
// and only if a segment asks; either may be nil.
type Env struct {
	Payload    payload.Payload
	Config     *config.Config
	Theme      theme.Theme
	Now        time.Time
	Columns    int
	Home       string
	Transcript func() transcript.Info
	RateLimits func() ratecache.Limits

	info   func() *transcript.Info
	limits func() *ratecache.Limits
}

// NewEnv returns the environment for one render.
func NewEnv(e Env) *Env {
	e.info = once(e.Transcript)
	e.limits = once(e.RateLimits)
	return &e
}

func once[T any](f func() T) func() *T {
	return sync.OnceValue(func() *T {
		var v T
		if f != nil {
			v = f()
		}
		return &v
	})
}

// Separator is the painted glyph drawn between segments, with a space either side.
func (e *Env) Separator() string {
	return " " + e.Theme.Paint(e.Theme.Muted, e.Config.String("separator")) + " "
}

// Lookup builds the segment with the given name, with the layout group it belongs to.
func (e *Env) Lookup(name string) (Segment, layout.Group, bool) {
	for _, r := range e.registrations() {
		if r.name == name {
			return r.build(e.Config.Section(name)), r.group, true
		}
	}
	return nil, 0, false
}

// All builds every segment, in the order they are listed here.
func (e *Env) All() []Segment {
	regs := e.registrations()
	all := make([]Segment, 0, len(regs))
	for _, r := range regs {
		all = append(all, r.build(e.Config.Section(r.name)))
	}
	return all
}

type registration struct {
	name  string
	group layout.Group
	build func(cfg *config.Section) Segment
}

// registrations is the one list of segments. To add a plugin, add its entries here.
func (e *Env) registrations() []registration {
	meter := func(cfg *config.Section) *bar.Meter {
		return &bar.Meter{Theme: e.Theme, Width: bar.WidthOf(e.Columns, cfg)}
	}
	rates := func(cfg *config.Section) rate.Deps {
		return rate.Deps{
			Config: cfg,
			Paint:  &e.Theme,
			Meter:  meter(cfg),
			Now:    e.Now,
			Limits: func() rate.Limits { return e.limits() },
		}
	}
	names := diagnostics.Names()
	regs := make([]registration, 0, 7+len(names))
	regs = append(regs, []registration{
		{"path", layout.Identity, func(cfg *config.Section) Segment {
			return path.New(path.Deps{Session: &e.Payload, Config: cfg, Paint: &e.Theme, Git: &gitbranch.Reader{}, Home: e.Home})
		}},
		{"model", layout.Identity, func(cfg *config.Section) Segment {
			return model.New(model.Deps{
				Session:    &e.Payload,
				Transcript: func() model.Transcript { return e.info() },
				Config:     cfg,
				Paint:      &e.Theme,
			})
		}},
		{"ctx", layout.Identity, func(cfg *config.Section) Segment {
			return ctx.New(ctx.Deps{Session: &e.Payload, Config: cfg, Paint: &e.Theme, Meter: meter(cfg)})
		}},
		{"tokens", layout.Usage, func(*config.Section) Segment {
			return tokens.New(tokens.Deps{Session: &e.Payload, Paint: &e.Theme, Separator: e.Separator()})
		}},
		{"cache", layout.Usage, func(cfg *config.Section) Segment {
			return cache.New(cache.Deps{Session: &e.Payload, Config: cfg, Paint: &e.Theme})
		}},
		{"rate5h", layout.Rates, func(cfg *config.Section) Segment { return rate.NewFiveHour(rates(cfg)) }},
		{"rate7d", layout.Rates, func(cfg *config.Section) Segment { return rate.NewSevenDay(rates(cfg)) }},
	}...)
	for _, name := range names {
		regs = append(regs, registration{name, layout.Usage, func(*config.Section) Segment {
			d, _ := diagnostics.New(name, diagnostics.Deps{
				Transcript: func() diagnostics.Transcript { return e.info() },
				Paint:      &e.Theme,
				Tokens:     tokens.FormatTokens,
			})
			return d
		}})
	}
	return regs
}

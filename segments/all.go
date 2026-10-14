// Package segments is the composition root of the status line: the one list of plugins it can show, and the one place
// that connects them to the rest of the program. Plugins declare what they need as interfaces in their own packages,
// and the small adapter types here satisfy those interfaces from the program's internal types.
package segments

import (
	_ "embed"
	"strconv"
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

var (
	//go:embed path/defaults.conf
	pathDefaults string
	//go:embed model/defaults.conf
	modelDefaults string
	//go:embed ctx/defaults.conf
	ctxDefaults string
	//go:embed cache/defaults.conf
	cacheDefaults string
	//go:embed rate/defaults.conf
	rateDefaults string
)

// Segment is what every plugin's segment provides.
type Segment interface {
	Name() string
	// Help says in a sentence or two what the segment shows and what its colors mean,
	// using the user's current settings for any threshold or label.
	Help() string
	// Keys lists every config key Render reads, relative to the segment's section, so a test can check them against defaults.conf.
	Keys() []string
	// Render returns the segment's text, or "" to hide it.
	Render() string
}

// Defaults returns the text of every plugin's defaults.conf, for layering under the user's config.
func Defaults() []string {
	return []string{pathDefaults, modelDefaults, ctxDefaults, cacheDefaults, rateDefaults}
}

// FormatTokens abbreviates a token count the way the tokens segment does.
func FormatTokens(n int64) string { return tokens.FormatTokens(n) }

// Sources is where a render gets its data. Transcript and RateLimits are called at most once per render, and only if a segment asks.
type Sources struct {
	Payload    payload.Payload
	Config     *config.Config
	Theme      theme.Theme
	Now        time.Time
	Columns    int
	Home       string
	Transcript func() transcript.Info
	RateLimits func() ratecache.Limits
}

// Env is one render's world, from which the segments are built.
type Env struct {
	Config  *config.Config
	Theme   theme.Theme
	Columns int

	src        *Sources
	transcript func() transcript.Info
	rateLimits func() ratecache.Limits
}

// NewEnv returns the environment for one render.
func NewEnv(src *Sources) *Env {
	return &Env{
		Config:     src.Config,
		Theme:      src.Theme,
		Columns:    src.Columns,
		src:        src,
		transcript: sync.OnceValue(orZero(src.Transcript)),
		rateLimits: sync.OnceValue(orZero(src.RateLimits)),
	}
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

// ContextBar draws the ctx segment for a usage percentage, which is how the subagent rows show the same bar.
func ContextBar(e *Env, usedPercentage float64) string {
	cfg := e.Config.Section("ctx")
	m := e.meter(cfg)
	m.fillPercent = true
	return ctx.New(ctx.Deps{Session: &reported{pct: usedPercentage}, Config: cfg, Paint: e.painter(), Meter: m}).Render()
}

type registration struct {
	name  string
	group layout.Group
	build func(cfg *config.Section) Segment
}

// registrations is the one list of segments. To add a plugin, add its entries here.
func (e *Env) registrations() []registration {
	names := diagnostics.Names()
	regs := make([]registration, 0, 7+len(names))
	regs = append(regs, []registration{
		{"path", layout.Identity, func(cfg *config.Section) Segment {
			return path.New(path.Deps{Session: e.session(), Config: cfg, Paint: e.painter(), Git: &gitReader{}, Home: e.src.Home})
		}},
		{"model", layout.Identity, func(cfg *config.Section) Segment {
			return model.New(model.Deps{
				Session:    e.session(),
				Transcript: func() model.Transcript { return &transcriptView{info: e.transcript()} },
				Config:     cfg,
				Paint:      e.painter(),
			})
		}},
		{"ctx", layout.Identity, func(cfg *config.Section) Segment { return e.context(e.session(), cfg) }},
		{"tokens", layout.Usage, func(*config.Section) Segment {
			return tokens.New(tokens.Deps{Session: e.session(), Paint: e.painter(), Separator: e.Separator()})
		}},
		{"cache", layout.Usage, func(cfg *config.Section) Segment {
			return cache.New(cache.Deps{Session: e.session(), Config: cfg, Paint: e.painter()})
		}},
		{"rate5h", layout.Rates, func(cfg *config.Section) Segment { return rate.NewFiveHour(e.rateDeps(cfg)) }},
		{"rate7d", layout.Rates, func(cfg *config.Section) Segment { return rate.NewSevenDay(e.rateDeps(cfg)) }},
	}...)
	for _, name := range names {
		regs = append(regs, registration{name, layout.Usage, func(*config.Section) Segment {
			d, _ := diagnostics.New(name, diagnostics.Deps{
				Transcript: func() diagnostics.Transcript { return &transcriptView{info: e.transcript()} },
				Paint:      e.painter(),
				Tokens:     tokens.FormatTokens,
			})
			return d
		}})
	}
	return regs
}

func (e *Env) context(session ctx.Session, cfg *config.Section) *ctx.Context {
	return ctx.New(ctx.Deps{Session: session, Config: cfg, Paint: e.painter(), Meter: e.meter(cfg)})
}

func (e *Env) rateDeps(cfg *config.Section) rate.Deps {
	return rate.Deps{
		Config: cfg,
		Paint:  e.painter(),
		Meter:  e.meter(cfg),
		Now:    e.src.Now,
		Limits: func() rate.Limits { return &limitsView{limits: e.rateLimits()} },
	}
}

func (e *Env) session() *session { return &session{p: &e.src.Payload} }

func (e *Env) painter() *painter { return &painter{th: e.Theme} }

func (e *Env) meter(cfg *config.Section) *meter {
	return &meter{th: e.Theme, width: bar.WidthOf(e.Columns, cfg)}
}

func orZero[T any](f func() T) func() T {
	if f == nil {
		return func() T { var zero T; return zero }
	}
	return f
}

// session answers the plugins' questions about the session from the payload.
type session struct{ p *payload.Payload }

func (s *session) DisplayName() string    { return s.p.Model.DisplayName }
func (s *session) ModelID() string        { return s.p.Model.ID }
func (s *session) EffortLevel() string    { return s.p.Effort.Level }
func (s *session) TranscriptPath() string { return s.p.TranscriptPath }
func (s *session) Cwd() string            { return s.p.Cwd }

func (s *session) LatestRequest() (input, cacheCreation, cacheRead int64) {
	u := s.p.ContextWindow.Current
	return u.Input, u.CacheCreation, u.CacheRead
}

func (s *session) TotalInput() (int64, bool)  { return deref(s.p.ContextWindow.TotalInput) }
func (s *session) TotalOutput() (int64, bool) { return deref(s.p.ContextWindow.TotalOutput) }

func (s *session) UsedPercentage() (float64, bool) { return deref(s.p.ContextWindow.UsedPercentage) }

func deref[T any](v *T) (T, bool) {
	if v == nil {
		var zero T
		return zero, false
	}
	return *v, true
}

// reported is a context usage that is already known, as the subagent rows have it.
type reported struct{ pct float64 }

func (r *reported) UsedPercentage() (float64, bool) { return r.pct, true }

// painter colors text with the theme's roles.
type painter struct{ th theme.Theme }

func (p *painter) Text(s string) string { return p.th.Paint(p.th.Text, s) }
func (p *painter) Warn(s string) string { return p.th.Paint(p.th.Warn, s) }
func (p *painter) Bad(s string) string  { return p.th.Paint(p.th.Bad, s) }
func (p *painter) Path(s string) string { return p.th.Paint(p.th.Path, s) }

// meter draws bars of one width in the theme.
type meter struct {
	th    theme.Theme
	width int
	// fillPercent colors the percentage with the fill's color even on a theme that colors its bar by position, as the subagent rows do.
	fillPercent bool
}

func (m *meter) Usage(usage, warn, bad int) string {
	return m.draw(&bar.Params{Usage: usage, Width: m.width, Warn: warn, Bad: bad, NoTarget: true})
}

func (m *meter) Pace(usage, target, warn, bad, hardCap int) string {
	return m.draw(&bar.Params{Usage: usage, Target: target, Width: m.width, Warn: warn, Bad: bad, HardCap: hardCap})
}

func (m *meter) draw(params *bar.Params) string {
	color := params.PercentColor(m.th)
	if m.fillPercent {
		color = params.Fill(m.th)
	}
	return params.Render(m.th) + " " + m.th.Paint(color, strconv.Itoa(params.Usage)+"%")
}

// transcriptView answers the plugins' questions about the session transcript.
type transcriptView struct{ info transcript.Info }

func (t *transcriptView) ActualModel() string      { return t.info.ActualModel }
func (t *transcriptView) Boot() int64              { return t.info.Boot }
func (t *transcriptView) SkillsChars() int         { return t.info.SkillsChars }
func (t *transcriptView) SkillsBare() int          { return t.info.SkillsBare }
func (t *transcriptView) SkillsTotal() int         { return t.info.SkillsTotal }
func (t *transcriptView) SubagentSkillsBare() int  { return t.info.SubBare }
func (t *transcriptView) SubagentSkillsTotal() int { return t.info.SubTotal }
func (t *transcriptView) MemDropped() bool         { return t.info.MemDropped }

// limitsView answers the rate plugin's questions about the rate-limit readings.
type limitsView struct{ limits ratecache.Limits }

func (l *limitsView) FiveHour() (rate.Reading, bool) { return reading(l.limits.FiveHour) }
func (l *limitsView) SevenDay() (rate.Reading, bool) { return reading(l.limits.SevenDay) }

func reading(r *ratecache.Reading) (rate.Reading, bool) {
	if r == nil {
		return rate.Reading{}, false
	}
	return rate.Reading{Used: r.Used, ResetsAt: r.ResetsAt}, true
}

// gitReader finds branches by reading .git files.
type gitReader struct{}

func (g *gitReader) Branch(dir string) string { return payload.Sanitize(gitbranch.Read(dir)) }

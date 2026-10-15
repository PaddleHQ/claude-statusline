// Package diagnostics provides the segments drawn from the session transcript: boot, skills, skills_stripped, agent_skills and mem.
package diagnostics

import "strconv"

const charsPerToken = 4

// tokenFn abbreviates a token count.
type tokenFn = func(n int64) string

//go:generate go run github.com/vektra/mockery/v2 --name=Transcript

// Transcript is what the segments need from the session transcript; every count is zero when the transcript has none.
type Transcript interface {
	BootTokens() int64
	SkillListChars() int
	// SkillCounts is how many skills in the initial listing lost their description, and how many are listed.
	SkillCounts() (bare, total int)
	// SubagentSkillCounts is the same for the subagent with the most stripped skills.
	SubagentSkillCounts() (bare, total int)
	MemoryDropped() bool
}

//go:generate go run github.com/vektra/mockery/v2 --name=Painter

// Painter colors text.
type Painter interface {
	Text(s string) string
	Warn(s string) string
	Bad(s string) string
}

// Deps is everything a segment is built from. Transcript is a function so that the transcript is only read when a segment is drawn.
// Tokens abbreviates a token count.
type Deps struct {
	Transcript func() Transcript
	Paint      Painter
	Tokens     tokenFn
}

// spec describes one diagnostic: show returns "" when there is nothing to report, and paint picks the text color.
type spec struct {
	name  string
	show  func(t Transcript, tokens tokenFn) string
	paint func(p Painter, s string) string
}

func text(p Painter, s string) string { return p.Text(s) }
func bad(p Painter, s string) string  { return p.Bad(s) }
func warn(p Painter, s string) string { return p.Warn(s) }
func counts(bare, total int) string {
	return strconv.Itoa(bare) + "/" + strconv.Itoa(total)
}

var specs = []spec{
	{"boot", func(t Transcript, tokens tokenFn) string {
		boot := t.BootTokens()
		if boot <= 0 {
			return ""
		}
		return "⌂ " + tokens(boot)
	}, text},
	{"skills", func(t Transcript, tokens tokenFn) string {
		chars := t.SkillListChars()
		if chars <= 0 {
			return ""
		}
		return "⚙ " + tokens(int64(chars/charsPerToken))
	}, text},
	{"skills_stripped", func(t Transcript, _ tokenFn) string {
		bare, total := t.SkillCounts()
		if bare <= 0 {
			return ""
		}
		return "✗ skills " + counts(bare, total)
	}, bad},
	{"agent_skills", func(t Transcript, _ tokenFn) string {
		bare, total := t.SubagentSkillCounts()
		if bare <= 0 {
			return ""
		}
		return "✗ agent skills " + counts(bare, total)
	}, warn},
	{"mem", func(t Transcript, _ tokenFn) string {
		if !t.MemoryDropped() {
			return ""
		}
		return "✗ mem"
	}, bad},
}

// Names lists the segments the package provides, which are the names New accepts.
func Names() []string {
	names := make([]string, 0, len(specs))
	for i := range specs {
		names = append(names, specs[i].name)
	}
	return names
}

// Diagnostic is one segment drawn from the session transcript.
type Diagnostic struct {
	d    Deps
	spec *spec
}

// New builds the named segment, and reports false when the package does not provide it.
func New(name string, d Deps) (*Diagnostic, bool) {
	for i := range specs {
		if specs[i].name == name {
			return &Diagnostic{d: d, spec: &specs[i]}, true
		}
	}
	return nil, false
}

// Name is the segment's name in the segments list.
func (g *Diagnostic) Name() string { return g.spec.name }

// Keys lists the config keys the segments read; they have none.
func (g *Diagnostic) Keys() []string { return nil }

// Render returns the segment's text, or "" to hide it.
func (g *Diagnostic) Render() string {
	out := g.spec.show(g.d.Transcript(), g.d.Tokens)
	if out == "" {
		return ""
	}
	return g.spec.paint(g.d.Paint, out)
}

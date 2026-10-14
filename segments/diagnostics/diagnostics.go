// Package diagnostics provides the segments drawn from the session transcript: boot, skills, skills_stripped, agent_skills and mem.
package diagnostics

import (
	"strconv"

	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/internal/transcript"
	"github.com/PaddleHQ/claude-statusline/segments/tokens"
)

const charsPerToken = 4

// spec describes one diagnostic: show returns "" when there is nothing to report, and color picks the text color.
type spec struct {
	name  string
	help  string
	show  func(transcript.Info) string
	color func(*theme.Theme) string
}

func text(th *theme.Theme) string { return th.Text }
func bad(th *theme.Theme) string  { return th.Bad }
func warn(th *theme.Theme) string { return th.Warn }
func counts(bare, total int) string {
	return strconv.Itoa(bare) + "/" + strconv.Itoa(total)
}

var specs = []spec{
	{"boot", "The prompt size on the first turn, before you said anything.", func(t transcript.Info) string {
		if t.Boot <= 0 {
			return ""
		}
		return "⌂ " + tokens.FormatTokens(t.Boot)
	}, text},
	{"skills", "The size of the always-loaded skill listing, in tokens.", func(t transcript.Info) string {
		if t.SkillsChars <= 0 {
			return ""
		}
		return "⚙ " + tokens.FormatTokens(int64(t.SkillsChars/charsPerToken))
	}, text},
	{"skills_stripped", "Red when skills lost their descriptions to fit the budget; shows how many.", func(t transcript.Info) string {
		if t.SkillsBare <= 0 {
			return ""
		}
		return "✗ skills " + counts(t.SkillsBare, t.SkillsTotal)
	}, bad},
	{"agent_skills", "Amber when a subagent lost skill descriptions; shows how many.", func(t transcript.Info) string {
		if t.SubBare <= 0 {
			return ""
		}
		return "✗ agent skills " + counts(t.SubBare, t.SubTotal)
	}, warn},
	{"mem", "Red when a SessionStart hook output was too large to inject.", func(t transcript.Info) string {
		if !t.MemDropped {
			return ""
		}
		return "✗ mem"
	}, bad},
}

// Plugin is the diagnostics plugin. It has no settings, so it has no defaults.conf.
func Plugin() *plugin.Plugin {
	entries := make([]plugin.Entry, 0, len(specs))
	for i := range specs {
		sp := &specs[i]
		entries = append(entries, plugin.Entry{Name: sp.name, New: func(p *plugin.Providers) plugin.Segment {
			return &diagnostic{p: p, spec: sp}
		}})
	}
	return &plugin.Plugin{Segments: entries}
}

// diagnostic is a usage-group segment drawn from the session transcript.
type diagnostic struct {
	p    *plugin.Providers
	spec *spec
}

func (d *diagnostic) Name() string { return d.spec.name }

func (d *diagnostic) Group() layout.Group { return layout.Usage }

func (d *diagnostic) Help() string { return d.spec.help }

func (d *diagnostic) Keys() []string { return nil }

func (d *diagnostic) Render(*plugin.Session) string {
	out := d.spec.show(d.p.Transcript())
	if out == "" {
		return ""
	}
	return d.p.Theme.Paint(d.spec.color(&d.p.Theme), out)
}

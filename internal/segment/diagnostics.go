package segment

import (
	"strconv"

	"github.com/PaddleHQ/claude-statusline/internal/layout"
)

const charsPerToken = 4

// diagnostic is a usage-group segment drawn from the session transcript; show returns "" when there is nothing to report.
type diagnostic struct {
	name   string
	show   func(*Context) string
	colour func(*Context) string
}

func (d diagnostic) Name() string { return d.name }

func (diagnostic) Group() layout.Group { return layout.Usage }

func (diagnostic) Keys() []string { return nil }

func (d diagnostic) Render(ctx *Context) string {
	text := d.show(ctx)
	if text == "" {
		return ""
	}
	return ctx.Theme.Paint(d.colour(ctx), text)
}

func text(ctx *Context) string { return ctx.Theme.Text }
func bad(ctx *Context) string  { return ctx.Theme.Bad }
func warn(ctx *Context) string { return ctx.Theme.Warn }
func counts(bare, total int) string {
	return strconv.Itoa(bare) + "/" + strconv.Itoa(total)
}

var diagnostics = []diagnostic{
	{"boot", func(c *Context) string {
		if c.Transcript.Boot <= 0 {
			return ""
		}
		return "⌂ " + FormatTokens(c.Transcript.Boot)
	}, text},
	{"skills", func(c *Context) string {
		if c.Transcript.SkillsChars <= 0 {
			return ""
		}
		return "⚙ " + FormatTokens(int64(c.Transcript.SkillsChars/charsPerToken))
	}, text},
	{"skills_stripped", func(c *Context) string {
		if c.Transcript.SkillsBare <= 0 {
			return ""
		}
		return "✗ skills " + counts(c.Transcript.SkillsBare, c.Transcript.SkillsTotal)
	}, bad},
	{"agent_skills", func(c *Context) string {
		if c.Transcript.SubBare <= 0 {
			return ""
		}
		return "✗ agent skills " + counts(c.Transcript.SubBare, c.Transcript.SubTotal)
	}, warn},
	{"mem", func(c *Context) string {
		if !c.Transcript.MemDropped {
			return ""
		}
		return "✗ mem"
	}, bad},
}

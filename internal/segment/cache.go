package segment

import (
	"strconv"

	"github.com/PaddleHQ/claude-statusline/internal/layout"
)

const percentScale = 100

// cache shows how much of the latest request's input came from the prompt cache.
type cache struct{}

func (c *cache) Name() string { return "cache" }

func (c *cache) Group() layout.Group { return layout.Usage }

func (c *cache) Keys() []string { return []string{"cache.warn"} }

func (c *cache) Render(ctx *Context) string {
	u := ctx.Payload.ContextWindow.Current
	total := u.Input + u.CacheCreation + u.CacheRead
	if total <= 0 {
		return ""
	}
	pct := u.CacheRead * percentScale / total
	color := ctx.Theme.Text
	if pct < int64(ctx.Config.Int("cache.warn")) {
		color = ctx.Theme.Bad
	}
	return ctx.Theme.Paint(color, "⧉ "+strconv.FormatInt(pct, 10)+"%")
}

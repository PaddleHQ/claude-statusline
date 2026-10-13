package segment

import "strconv"

const percentScale = 100

// cache shows how much of the latest request's input came from the prompt cache.
type cache struct{}

func (cache) Name() string { return "cache" }

func (cache) Keys() []string { return []string{"cache.warn"} }

func (cache) Render(ctx *Context) string {
	u := ctx.Payload.ContextWindow.Current
	total := u.Input + u.CacheCreation + u.CacheRead
	if total <= 0 {
		return ""
	}
	pct := u.CacheRead * percentScale / total
	colour := ctx.Theme.Text
	if pct < int64(ctx.Config.Int("cache.warn")) {
		colour = ctx.Theme.Bad
	}
	return ctx.Theme.Paint(colour, "⧉ "+strconv.FormatInt(pct, 10)+"%")
}

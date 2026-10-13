package segment

import (
	"math"
	"strconv"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
)

// ctxBar draws context-window usage as "ctx: <bar> <NN>%", coloured by absolute usage.
type ctxBar struct{}

func (ctxBar) Name() string { return "ctx" }

func (ctxBar) Group() layout.Group { return layout.Identity }

func (ctxBar) Keys() []string { return []string{"ctx.width", "ctx.warn", "ctx.bad"} }

func (ctxBar) Render(ctx *Context) string {
	used := ctx.Payload.ContextWindow.UsedPercentage
	if used == nil {
		return ""
	}
	cfg := ctx.Config
	pct := int(math.RoundToEven(*used))
	b := bar.Render(bar.Params{
		Usage:    pct,
		Width:    bar.ResponsiveWidth(cfg.Int("ctx.width"), ctx.Columns),
		Warn:     cfg.Int("ctx.warn"),
		Bad:      cfg.Int("ctx.bad"),
		NoTarget: true,
	}, ctx.Theme)
	return ctx.Theme.Paint(ctx.Theme.Text, "ctx: ") + b + " " + strconv.Itoa(pct) + "%"
}

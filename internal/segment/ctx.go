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

func (ctxBar) Keys() []string {
	return []string{"ctx.width", "ctx.width_medium", "ctx.width_narrow", "ctx.warn", "ctx.bad"}
}

func (ctxBar) Render(ctx *Context) string {
	used := ctx.Payload.ContextWindow.UsedPercentage
	if used == nil {
		return ""
	}
	cfg := ctx.Config
	pct := int(math.RoundToEven(*used))
	params := bar.Params{
		Usage:    pct,
		Width:    barWidth(ctx, "ctx"),
		Warn:     cfg.Int("ctx.warn"),
		Bad:      cfg.Int("ctx.bad"),
		NoTarget: true,
	}
	b := bar.Render(params, ctx.Theme)
	return ctx.Theme.Paint(ctx.Theme.Text, "ctx: ") + b + " " + ctx.Theme.Paint(bar.PercentColor(params, ctx.Theme), strconv.Itoa(pct)+"%")
}

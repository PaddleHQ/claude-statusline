package segment

import (
	"fmt"
	"strings"
)

const (
	thousand = 1000
	million  = 1000 * thousand
	tenth    = 10
)

// tokens shows session token totals as "↓ <in> · ↑ <out>".
type tokens struct{}

func (tokens) Name() string { return "tokens" }

func (tokens) Keys() []string { return nil }

func (tokens) Render(ctx *Context) string {
	cw := ctx.Payload.ContextWindow
	var parts []string
	if cw.TotalInput != nil {
		parts = append(parts, ctx.Theme.Paint(ctx.Theme.Text, "↓ "+formatTokens(*cw.TotalInput)))
	}
	if cw.TotalOutput != nil {
		parts = append(parts, ctx.Theme.Paint(ctx.Theme.Text, "↑ "+formatTokens(*cw.TotalOutput)))
	}
	return strings.Join(parts, separator(ctx))
}

// formatTokens abbreviates a count, truncating: 1500 is 1.5k and 1500000 is 1.5M.
func formatTokens(n int64) string {
	switch {
	case n >= million:
		return fmt.Sprintf("%d.%dM", n/million, n%million/(million/tenth))
	case n >= thousand:
		return fmt.Sprintf("%d.%dk", n/thousand, n%thousand/(thousand/tenth))
	}
	return fmt.Sprintf("%d", n)
}

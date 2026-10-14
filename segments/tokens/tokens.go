// Package tokens provides the tokens segment: session token totals.
package tokens

import (
	"fmt"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
)

const (
	name     = "tokens"
	thousand = 1000
	million  = 1000 * thousand
	tenth    = 10
)

// Plugin is the tokens plugin. It has no settings of its own, so it has no defaults.conf.
func Plugin() *plugin.Plugin {
	return &plugin.Plugin{Segments: []plugin.Entry{{Name: name, New: New}}}
}

// New builds the tokens segment.
func New(p *plugin.Providers) plugin.Segment { return &tokens{p: p} }

// tokens shows session token totals as "↓ <in> · ↑ <out>".
type tokens struct{ p *plugin.Providers }

func (t *tokens) Name() string { return name }

func (t *tokens) Group() layout.Group { return layout.Usage }

func (t *tokens) Help() string {
	return "Session token totals: input after the down arrow, output after the up arrow."
}

func (t *tokens) Keys() []string { return nil }

func (t *tokens) Render(s *plugin.Session) string {
	th := t.p.Theme
	cw := s.Payload.ContextWindow
	var parts []string
	if cw.TotalInput != nil {
		parts = append(parts, th.Paint(th.Text, "↓ "+FormatTokens(*cw.TotalInput)))
	}
	if cw.TotalOutput != nil {
		parts = append(parts, th.Paint(th.Text, "↑ "+FormatTokens(*cw.TotalOutput)))
	}
	return strings.Join(parts, t.p.Separator())
}

// FormatTokens abbreviates a count, truncating: 1500 is 1.5k and 1500000 is 1.5M.
func FormatTokens(n int64) string {
	switch {
	case n >= million:
		return fmt.Sprintf("%d.%dM", n/million, n%million/(million/tenth))
	case n >= thousand:
		return fmt.Sprintf("%d.%dk", n/thousand, n%thousand/(thousand/tenth))
	}
	return fmt.Sprintf("%d", n)
}

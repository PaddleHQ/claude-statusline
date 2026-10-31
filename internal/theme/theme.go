// Package theme defines the colors and glyphs a status line is drawn with.
package theme

import "fmt"

const (
	rampStops   = 3
	rampScale   = 1000
	rampSpan    = 2 * rampScale
	rampGoodAt  = 0
	rampWarnAt  = 1
	rampBadAt   = 2
	channelsRGB = 3
	glowScale   = 100
	whiteChan   = 255
)

// Theme holds role colors as raw escape sequences and the glyphs for bars.
// An empty color means "leave the terminal's own".
type Theme struct {
	TextColor string
	Muted     string
	Good      string
	WarnColor string
	BadColor  string
	Marker    string
	PathColor string
	Reset     string

	BranchColor   string
	WorktreeColor string

	Full        string
	Partials    [7]string
	Empty       string
	MarkerGlyph string

	BranchGlyph   string
	WorktreeGlyph string

	// Gradient draws whole cells colored by position along Ramp, with the
	// pace color on the percentage text. Ramp holds the good, warn and bad
	// stops as RGB, and is nil once color is removed.
	Gradient    bool
	Ramp        [][channelsRGB]int
	EmptyColor  string
	MarkerColor string

	// Saber draws a lightsaber blade: the whole blade takes the Blade color for
	// the pace state (good, warn, bad), brightening towards white by up to Glow
	// percent at the tip over its last GlowCells cells, and Tip is the glyph for a half-lit final cell. Blade
	// is nil once color is removed.
	Saber bool
	Blade [][channelsRGB]int
	Glow  int
	// GlowCells is how many of the last lit cells glow.
	GlowCells int
	Tip       string
}

// WithoutColor clears every color and style, keeping the glyphs.
func (t *Theme) WithoutColor() Theme {
	c := *t
	c.TextColor, c.Muted, c.Good, c.WarnColor, c.BadColor, c.Marker, c.PathColor, c.Reset = "", "", "", "", "", "", "", ""
	c.BranchColor, c.WorktreeColor = "", ""
	c.Ramp, c.Blade, c.EmptyColor, c.MarkerColor = nil, nil, "", ""
	return c
}

// Paint wraps s in the color and a reset, or returns it untouched when the color is empty.
func (t *Theme) Paint(color, s string) string {
	if color == "" {
		return s
	}
	return color + s + t.Reset
}

// Text paints s in the text color.
func (t *Theme) Text(s string) string { return t.Paint(t.TextColor, s) }

// Warn paints s in the warning color.
func (t *Theme) Warn(s string) string { return t.Paint(t.WarnColor, s) }

// Bad paints s in the bad color.
func (t *Theme) Bad(s string) string { return t.Paint(t.BadColor, s) }

// Path paints s in the path color.
func (t *Theme) Path(s string) string { return t.Paint(t.PathColor, s) }

// Branch paints s in the branch color.
func (t *Theme) Branch(s string) string { return t.Paint(t.BranchColor, s) }

// Worktree paints s in the worktree color.
func (t *Theme) Worktree(s string) string { return t.Paint(t.WorktreeColor, s) }

// Glyphs returns the branch and worktree glyphs.
func (t *Theme) Glyphs() (branch, worktree string) { return t.BranchGlyph, t.WorktreeGlyph }

// RampColor is the escape sequence for a cell of a width-cell gradient bar, or
// "" when the theme has no ramp. The first cell takes the good stop, the middle
// the warn stop and the last the bad stop, with integer steps between.
func (t *Theme) RampColor(cell, width int) string {
	if len(t.Ramp) != rampStops {
		return ""
	}
	pos := 0
	if width > 1 {
		pos = cell * rampSpan / (width - 1)
	}
	from, to := t.Ramp[rampGoodAt], t.Ramp[rampWarnAt]
	if pos > rampScale {
		pos -= rampScale
		from, to = t.Ramp[rampWarnAt], t.Ramp[rampBadAt]
	}
	var c [channelsRGB]int
	for i := range c {
		c[i] = from[i] + (to[i]-from[i])*pos/rampScale
	}
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", c[0], c[1], c[2])
}

// BladeColor is the escape sequence for lit cell cell of a saber blade of lit
// cells in the given pace state (0 good, 1 warn, 2 bad), or "" when the theme
// has no blade. Only the last GlowCells lit cells glow: the j-th of k cells in
// that window is blended Glow*j/k percent towards white, so the tip takes the
// full Glow, and every cell before the window is the plain state color.
func (t *Theme) BladeColor(state, cell, lit int) string {
	if len(t.Blade) != rampStops || state < rampGoodAt || state > rampBadAt {
		return ""
	}
	window := min(t.GlowCells, lit)
	glow := 0
	if j := cell - (lit - window) + 1; j > 0 {
		glow = t.Glow * j / window
	}
	base := t.Blade[state]
	var c [channelsRGB]int
	for i := range c {
		c[i] = base[i] + (whiteChan-base[i])*glow/glowScale
	}
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", c[0], c[1], c[2])
}

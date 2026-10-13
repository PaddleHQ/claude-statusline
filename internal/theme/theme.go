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
)

// Theme holds role colors as raw escape sequences and the glyphs for bars.
// An empty color means "leave the terminal's own".
type Theme struct {
	Text   string
	Muted  string
	Good   string
	Warn   string
	Bad    string
	Marker string
	Path   string
	Reset  string

	Full        string
	Partials    [7]string
	Empty       string
	MarkerGlyph string

	// Gradient draws whole cells colored by position along Ramp, with the
	// pace color on the percentage text. Ramp holds the good, warn and bad
	// stops as RGB, and is nil once color is removed.
	Gradient    bool
	Ramp        [][channelsRGB]int
	EmptyColor  string
	MarkerColor string
}

// WithoutColor clears every color and style, keeping the glyphs.
func (t Theme) WithoutColor() Theme {
	t.Text, t.Muted, t.Good, t.Warn, t.Bad, t.Marker, t.Path, t.Reset = "", "", "", "", "", "", "", ""
	t.Ramp, t.EmptyColor, t.MarkerColor = nil, "", ""
	return t
}

// Paint wraps s in the color and a reset, or returns it untouched when the color is empty.
func (t Theme) Paint(color, s string) string {
	if color == "" {
		return s
	}
	return color + s + t.Reset
}

// RampColor is the escape sequence for a cell of a width-cell gradient bar, or
// "" when the theme has no ramp. The first cell takes the good stop, the middle
// the warn stop and the last the bad stop, with integer steps between.
func (t Theme) RampColor(cell, width int) string {
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

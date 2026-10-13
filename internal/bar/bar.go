// Package bar draws a usage bar with a pace marker.
package bar

import (
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/theme"
)

const (
	maxWidth     = 200
	minWidth     = 4
	narrowCols   = 120
	mediumCols   = 160
	eighths      = 8
	percentScale = 100
)

// Params describes one bar. Usage and Target are percentages; Warn and Bad are
// deviations from the target in percentage points. With NoTarget set there is no
// marker, Target and HardCap are ignored, and Warn and Bad are usage percentages.
type Params struct {
	Usage    int
	Target   int
	Width    int
	Warn     int
	Bad      int
	HardCap  int
	NoTarget bool
}

// ResponsiveWidth scales the configured width down for narrow terminals.
func ResponsiveWidth(width, columns int) int {
	switch {
	case columns < narrowCols:
		width /= 2
	case columns < mediumCols:
		width = width * 7 / 10
	}
	return min(max(width, minWidth), maxWidth)
}

// ElapsedPercent is how much of a window, ending at resetsAt, has passed by now, clamped to 0..100.
func ElapsedPercent(now, resetsAt, windowSeconds int64) int {
	elapsed := min(max(now-(resetsAt-windowSeconds), 0), windowSeconds)
	return int(elapsed * percentScale / windowSeconds)
}

// Render draws the bar: a fill coloured by distance from the pace target, a marker
// on the target cell and muted empty cells. Without a target the fill is coloured
// by absolute usage and there is no marker.
func Render(p Params, th theme.Theme) string {
	maxEighths := p.Width * eighths
	filled := min((p.Usage*maxEighths+percentScale/2)/percentScale, maxEighths)
	fullCells, remainder := filled/eighths, filled%eighths

	targetCell := -1
	fill := th.Good
	if p.NoTarget {
		switch {
		case p.Usage >= p.Bad:
			fill = th.Bad
		case p.Usage >= p.Warn:
			fill = th.Warn
		}
	} else {
		targetCell = min(p.Target*p.Width/percentScale, p.Width-1)
		deviation := max(p.Usage-p.Target, p.Target-p.Usage)
		switch {
		case p.Usage >= p.HardCap || deviation > p.Bad:
			fill = th.Bad
		case deviation > p.Warn:
			fill = th.Warn
		}
	}

	var b strings.Builder
	for cell := range p.Width {
		switch {
		case cell == targetCell:
			b.WriteString(th.Paint(th.Marker, th.MarkerGlyph))
		case cell < fullCells:
			b.WriteString(th.Paint(fill, th.Full))
		case cell == fullCells && remainder > 0:
			b.WriteString(th.Paint(fill, th.Partials[remainder-1]))
		default:
			b.WriteString(th.Paint(th.Muted, th.Empty))
		}
	}
	return b.String()
}

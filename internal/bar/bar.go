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

// WidthFor picks the bar width for a terminal: narrow below 120 columns, medium below 160, otherwise wide.
func WidthFor(columns, wide, medium, narrow int) int {
	width := wide
	switch {
	case columns < narrowCols:
		width = narrow
	case columns < mediumCols:
		width = medium
	}
	return min(max(width, minWidth), maxWidth)
}

// ElapsedPercent is how much of a window, ending at resetsAt, has passed by now, clamped to 0..100.
func ElapsedPercent(now, resetsAt, windowSeconds int64) int {
	elapsed := min(max(now-(resetsAt-windowSeconds), 0), windowSeconds)
	return int(elapsed * percentScale / windowSeconds)
}

// Fill is the color for the bar's state: distance from the pace target, or absolute usage without a target.
func (p *Params) Fill(th theme.Theme) string {
	if p.NoTarget {
		switch {
		case p.Usage >= p.Bad:
			return th.Bad
		case p.Usage >= p.Warn:
			return th.Warn
		}
		return th.Good
	}
	deviation := max(p.Usage-p.Target, p.Target-p.Usage)
	switch {
	case p.Usage >= p.HardCap || deviation > p.Bad:
		return th.Bad
	case deviation > p.Warn:
		return th.Warn
	}
	return th.Good
}

// PercentColor is the color for the percentage printed after the bar. A gradient
// bar colors its cells by position, so the state color moves to the text.
func (p *Params) PercentColor(th theme.Theme) string {
	if th.Gradient {
		return p.Fill(th)
	}
	return ""
}

// Render draws the bar: a fill colored by distance from the pace target, a marker
// on the target cell and muted empty cells. Without a target the fill is colored
// by absolute usage and there is no marker. A gradient theme draws whole cells
// colored by position instead.
func (p *Params) Render(th theme.Theme) string {
	maxEighths := p.Width * eighths
	filled := min((p.Usage*maxEighths+percentScale/2)/percentScale, maxEighths)

	targetCell := -1
	if !p.NoTarget {
		targetCell = min(p.Target*p.Width/percentScale, p.Width-1)
	}
	if th.Gradient {
		return p.gradientCells(th, filled, targetCell)
	}
	return p.solidCells(th, filled, targetCell)
}

func (p *Params) gradientCells(th theme.Theme, filled, targetCell int) string {
	var b strings.Builder
	cells := (filled + eighths/2) / eighths
	for cell := range p.Width {
		switch {
		case cell == targetCell:
			b.WriteString(th.Paint(th.MarkerColor, th.MarkerGlyph))
		case cell < cells:
			b.WriteString(th.Paint(th.RampColor(cell, p.Width), th.Full))
		default:
			b.WriteString(th.Paint(th.EmptyColor, th.Empty))
		}
	}
	return b.String()
}

func (p *Params) solidCells(th theme.Theme, filled, targetCell int) string {
	fullCells, remainder := filled/eighths, filled%eighths
	fill := p.Fill(th)

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

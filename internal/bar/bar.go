// Package bar draws a usage bar with a pace marker.
package bar

import (
	"strconv"
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

// Private-use glyphs from Nerd Fonts: a capsule drawn as a left cap, a middle and a right cap, empty then filled.
const (
	capsuleEmptyFirst  = "\uee00"
	capsuleEmptyMiddle = "\uee01"
	capsuleEmptyLast   = "\uee02"
	capsuleFullFirst   = "\uee03"
	capsuleFullMiddle  = "\uee04"
	capsuleFullLast    = "\uee05"
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
	// Nerd says the terminal can draw Nerd Font glyphs, which a capsule theme needs.
	Nerd bool
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

// Sizes reads whole-number settings; a config section satisfies it.
type Sizes interface {
	Int(key string) int
}

// WidthOf picks the bar width for a terminal from the width, width_medium and width_narrow settings.
func WidthOf(columns int, s Sizes) int {
	return WidthFor(columns, s.Int("width"), s.Int("width_medium"), s.Int("width_narrow"))
}

// state classifies the bar: distance from the pace target, or absolute usage without a target.
func (p *Params) state() int {
	if p.NoTarget {
		switch {
		case p.Usage >= p.Bad:
			return theme.StateBad
		case p.Usage >= p.Warn:
			return theme.StateWarn
		}
		return theme.StateGood
	}
	deviation := max(p.Usage-p.Target, p.Target-p.Usage)
	switch {
	case p.Usage >= p.HardCap || deviation > p.Bad:
		return theme.StateBad
	case deviation > p.Warn:
		return theme.StateWarn
	}
	return theme.StateGood
}

// Fill is the color for the bar's state: distance from the pace target, or
// absolute usage without a target. A saber theme takes it from its blade.
func (p *Params) Fill(th theme.Theme) string {
	if th.Saber {
		return th.BladeColor(p.state(), 0, 0)
	}
	switch p.state() {
	case theme.StateBad:
		return th.BadColor
	case theme.StateWarn:
		return th.WarnColor
	}
	return th.Good
}

// PercentColor is the color for the percentage printed after the bar. A gradient
// bar colors its cells by position, a saber bar by blade, and a capsule bar with Nerd glyphs by
// pace, so the state color moves to the text.
func (p *Params) PercentColor(th theme.Theme) string {
	if th.Gradient || th.Saber || (th.Capsule && p.Nerd) {
		return p.Fill(th)
	}
	return ""
}

// Render draws the bar: a fill colored by distance from the pace target, a marker
// on the target cell and muted empty cells. Without a target the fill is colored
// by absolute usage and there is no marker. A gradient theme draws whole cells
// colored by position instead, a saber theme draws a blade in half cells, and a capsule theme
// draws rounded Nerd Font cells.
func (p *Params) Render(th theme.Theme) string {
	maxEighths := p.Width * eighths
	filled := min((p.Usage*maxEighths+percentScale/2)/percentScale, maxEighths)

	targetCell := -1
	if !p.NoTarget {
		targetCell = min(p.Target*p.Width/percentScale, p.Width-1)
	}
	switch {
	case th.Saber:
		return p.saberCells(th, targetCell)
	case th.Gradient:
		return p.gradientCells(th, filled, targetCell)
	case th.Capsule && p.Nerd:
		return p.capsuleCells(th, targetCell)
	}
	return p.solidCells(th, filled, targetCell)
}

func (p *Params) gradientCells(th theme.Theme, filled, targetCell int) string {
	var r runs
	cells := (filled + eighths/2) / eighths
	for cell := range p.Width {
		switch {
		case cell == targetCell:
			r.add(th.MarkerColor, th.MarkerGlyph)
		case cell < cells:
			r.add(th.RampColor(cell, p.Width), th.Full)
		default:
			r.add(th.EmptyColor, th.Empty)
		}
	}
	return r.paint(th)
}

// saberCells lights the blade in half cells; a lit cell brightens towards the tip.
func (p *Params) saberCells(th theme.Theme, targetCell int) string {
	halves := min((p.Usage*p.Width*2+percentScale/2)/percentScale, p.Width*2)
	fullCells, tip := halves/2, halves%2 == 1
	lit := fullCells
	if tip {
		lit++
	}
	state := p.state()

	var r runs
	for cell := range p.Width {
		switch {
		case cell == targetCell:
			r.add(th.MarkerColor, th.MarkerGlyph)
		case cell < fullCells:
			r.add(th.BladeColor(state, cell, lit), th.Full)
		case cell == fullCells && tip:
			r.add(th.BladeColor(state, cell, lit), th.Tip)
		default:
			r.add(th.EmptyColor, th.Empty)
		}
	}
	return r.paint(th)
}

// capsuleCells draws whole rounded cells in the state color. A cell's glyph depends only on its
// position and whether it is filled, so the marker keeps the filled shape and changes only color.
func (p *Params) capsuleCells(th theme.Theme, targetCell int) string {
	filled := min((p.Usage*p.Width+percentScale/2)/percentScale, p.Width)
	fill := p.Fill(th)

	var r runs
	for cell := range p.Width {
		isFilled := cell < filled || cell == targetCell
		glyph := capsuleGlyph(cell, p.Width, isFilled)
		switch {
		case cell == targetCell:
			r.add(th.MarkerColor, glyph)
		case isFilled:
			r.add(fill, glyph)
		default:
			r.add(th.EmptyColor, glyph)
		}
	}
	return r.paint(th)
}

func capsuleGlyph(cell, width int, filled bool) string {
	switch {
	case cell == 0 && filled:
		return capsuleFullFirst
	case cell == 0:
		return capsuleEmptyFirst
	case cell == width-1 && filled:
		return capsuleFullLast
	case cell == width-1:
		return capsuleEmptyLast
	case filled:
		return capsuleFullMiddle
	}
	return capsuleEmptyMiddle
}

func (p *Params) solidCells(th theme.Theme, filled, targetCell int) string {
	fullCells, remainder := filled/eighths, filled%eighths
	fill := p.Fill(th)

	var r runs
	for cell := range p.Width {
		switch {
		case cell == targetCell:
			r.add(th.Marker, th.MarkerGlyph)
		case cell < fullCells:
			r.add(fill, th.Full)
		case cell == fullCells && remainder > 0:
			r.add(fill, th.Partials[remainder-1])
		default:
			r.add(th.Muted, th.Empty)
		}
	}
	return r.paint(th)
}

type run struct {
	color string
	text  strings.Builder
}

// runs groups consecutive cells that share a color so each group is painted once.
type runs []*run

func (r *runs) add(color, glyph string) {
	if n := len(*r); n > 0 && (*r)[n-1].color == color {
		(*r)[n-1].text.WriteString(glyph)
		return
	}
	next := &run{color: color}
	next.text.WriteString(glyph)
	*r = append(*r, next)
}

func (r runs) paint(th theme.Theme) string {
	var b strings.Builder
	for _, g := range r {
		b.WriteString(th.Paint(g.color, g.text.String()))
	}
	return b.String()
}

// Meter draws bars of one width in a theme, each followed by its percentage.
type Meter struct {
	Theme theme.Theme
	Width int
	// FillPercent colors the percentage with the fill's color even on a theme that colors its bar by position.
	FillPercent bool
	// Nerd says the terminal can draw Nerd Font glyphs.
	Nerd bool
}

// Usage draws a bar with no pace marker; warn and bad are usage percentages at which the fill changes color.
func (m *Meter) Usage(usage, warn, bad int) string {
	return m.draw(&Params{Usage: usage, Width: m.Width, Warn: warn, Bad: bad, NoTarget: true})
}

// Pace draws a bar with a marker at the target; warn and bad are deviations from it, and the fill is bad from hardCap.
func (m *Meter) Pace(usage, target, warn, bad, hardCap int) string {
	return m.draw(&Params{Usage: usage, Target: target, Width: m.Width, Warn: warn, Bad: bad, HardCap: hardCap})
}

func (m *Meter) draw(params *Params) string {
	params.Nerd = m.Nerd
	color := params.PercentColor(m.Theme)
	if m.FillPercent {
		color = params.Fill(m.Theme)
	}
	return params.Render(m.Theme) + " " + m.Theme.Paint(color, strconv.Itoa(params.Usage)+"%")
}

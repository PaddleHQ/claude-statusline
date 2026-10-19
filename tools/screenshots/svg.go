package main

import (
	"fmt"
	"strings"
)

const (
	fontSize   = 15
	cellWidth  = 9
	lineHeight = 20
	baseline   = 15
	padding    = 16
	radius     = 8
	dimOpacity = "0.55"
	fontFamily = "ui-monospace, SFMono-Regular, Menlo, Consolas, 'DejaVu Sans Mono', 'Liberation Mono', monospace"

	cubeStart  = 16
	cubeSize   = 6
	grayStart  = 232
	grayBase   = 8
	grayStep   = 10
	cubeStep   = 40
	cubeOffset = 55
)

// Theme is the palette an image is drawn with.
type Theme struct {
	Name       string
	Background string
	Foreground string
	ANSI       [16]string
}

// Dark is drawn for GitHub's dark appearance.
var Dark = &Theme{
	Name:       "dark",
	Background: "#0d1117",
	Foreground: "#c9d1d9",
	ANSI: [16]string{
		"#484f58", "#ff7b72", "#3fb950", "#d29922", "#58a6ff", "#bc8cff", "#39c5cf", "#b1bac4",
		"#6e7681", "#ffa198", "#56d364", "#e3b341", "#79c0ff", "#d2a8ff", "#56d4dd", "#f0f6fc",
	},
}

// Light is drawn for GitHub's light appearance.
var Light = &Theme{
	Name:       "light",
	Background: "#ffffff",
	Foreground: "#24292f",
	ANSI: [16]string{
		"#24292f", "#cf222e", "#1a7f37", "#9a6700", "#0969da", "#8250df", "#1b7c83", "#6e7781",
		"#57606a", "#a40e26", "#2da44e", "#bf8700", "#218bff", "#a475f9", "#3192aa", "#8c959f",
	},
}

// run is a stretch of cells that share a style, starting at a column.
type run struct {
	col  int
	text []rune
	st   style
}

// SVG draws the screen on a monospace grid, one cell per rune, with the theme's background and colors.
func (s *Screen) SVG(th *Theme) []byte {
	cols := 0
	for _, line := range s.lines {
		cols = max(cols, len(line))
	}
	width := 2*padding + cols*cellWidth
	height := 2*padding + len(s.lines)*lineHeight

	var b strings.Builder
	fmt.Fprintf(&b, "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\" ", width, height)
	fmt.Fprintf(&b, "viewBox=\"0 0 %d %d\">\n", width, height)
	fmt.Fprintf(&b, "<rect width=\"%d\" height=\"%d\" rx=\"%d\" fill=\"%s\"/>\n", width, height, radius, th.Background)
	fmt.Fprintf(&b, "<g font-family=\"%s\" font-size=\"%d\" fill=\"%s\" xml:space=\"preserve\">\n", fontFamily, fontSize, th.Foreground)
	for i, line := range s.lines {
		y := padding + i*lineHeight + baseline
		fmt.Fprintf(&b, "<text y=\"%d\">", y)
		for _, r := range runsOf(line) {
			writeRun(&b, r, th)
		}
		b.WriteString("</text>\n")
	}
	b.WriteString("</g>\n</svg>\n")
	return []byte(b.String())
}

func runsOf(line []cell) []run {
	var runs []run
	for col, c := range line {
		if n := len(runs); n > 0 && runs[n-1].st == c.st {
			runs[n-1].text = append(runs[n-1].text, c.r)
			continue
		}
		runs = append(runs, run{col: col, text: []rune{c.r}, st: c.st})
	}
	return runs
}

func writeRun(b *strings.Builder, r run, th *Theme) {
	text := string(r.text)
	if strings.TrimSpace(text) == "" && !r.st.underline {
		return
	}
	fmt.Fprintf(b, "<tspan x=\"%d\"", padding+r.col*cellWidth)
	if hex := r.st.fg.hex(th); hex != "" {
		fmt.Fprintf(b, " fill=\"%s\"", hex)
	}
	if r.st.dim {
		fmt.Fprintf(b, " opacity=\"%s\"", dimOpacity)
	}
	if r.st.bold {
		b.WriteString(" font-weight=\"bold\"")
	}
	if r.st.italic {
		b.WriteString(" font-style=\"italic\"")
	}
	if r.st.underline {
		b.WriteString(" text-decoration=\"underline\"")
	}
	b.WriteString(">")
	b.WriteString(escapeXML(text))
	b.WriteString("</tspan>")
}

// hex resolves the color to #rrggbb, or an empty string for the theme's default foreground.
func (c color) hex(th *Theme) string {
	switch c.kind {
	case kindRGB:
		return fmt.Sprintf("#%02x%02x%02x", c.r, c.g, c.b)
	case kindIndexed:
		return indexedHex(c.index, th)
	default:
		return ""
	}
}

func indexedHex(n int, th *Theme) string {
	switch {
	case n < len(th.ANSI):
		return th.ANSI[n]
	case n < grayStart:
		n -= cubeStart
		return fmt.Sprintf("#%02x%02x%02x", cubeLevel(n/(cubeSize*cubeSize)), cubeLevel(n/cubeSize%cubeSize), cubeLevel(n%cubeSize))
	default:
		g := grayBase + (n-grayStart)*grayStep
		return fmt.Sprintf("#%02x%02x%02x", g, g, g)
	}
}

func cubeLevel(i int) int {
	if i == 0 {
		return 0
	}
	return cubeOffset + i*cubeStep
}

var xmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")

func escapeXML(s string) string {
	return xmlEscaper.Replace(s)
}

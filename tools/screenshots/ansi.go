package main

import (
	"strconv"
	"strings"
)

const (
	esc           = '\033'
	csi           = '['
	sgrFinal      = 'm'
	sgrReset      = 0
	sgrBold       = 1
	sgrDim        = 2
	sgrItalic     = 3
	sgrUnderline  = 4
	sgrFGBase     = 30
	sgrFGEnd      = 37
	sgrFGExtended = 38
	sgrBrightBase = 90
	sgrBrightEnd  = 97
	modeIndexed   = 5
	modeRGB       = 2
	brightOffset  = 8
)

type colorKind int

const (
	kindDefault colorKind = iota
	kindIndexed
	kindRGB
)

type color struct {
	kind    colorKind
	index   int
	r, g, b int
}

type style struct {
	bold, dim, italic, underline bool
	fg                           color
}

type cell struct {
	r  rune
	st style
}

// Screen is a grid of styled cells, one per rune, built from terminal output.
type Screen struct {
	lines [][]cell
}

// Parse reads terminal output into a Screen. It understands the escape sequences the status line emits:
// reset, bold, dim, italic, underline, the 16 basic foreground colors, 256-color and true color.
// Anything else is ignored.
func Parse(ansi string) *Screen {
	s := &Screen{lines: [][]cell{nil}}
	var st style
	runes := []rune(ansi)
	for i := 0; i < len(runes); i++ {
		switch r := runes[i]; {
		case r == esc && i+1 < len(runes) && runes[i+1] == csi:
			i = s.escape(runes, i+2, &st)
		case r == '\n':
			s.lines = append(s.lines, nil)
		case r < ' ' || r == 0x7f:
		default:
			last := len(s.lines) - 1
			s.lines[last] = append(s.lines[last], cell{r: r, st: st})
		}
	}
	return s
}

// escape handles the sequence whose parameters start at runes[from] and returns the index of its final byte.
func (s *Screen) escape(runes []rune, from int, st *style) int {
	end := from
	for end < len(runes) && (runes[end] == ';' || runes[end] >= '0' && runes[end] <= '9') {
		end++
	}
	if end < len(runes) && runes[end] == sgrFinal {
		st.apply(string(runes[from:end]))
	}
	return min(end, len(runes)-1)
}

func (st *style) apply(params string) {
	codes := parseCodes(params)
	for i := 0; i < len(codes); i++ {
		if codes[i] == sgrFGExtended {
			var used int
			st.fg, used = extendedColor(codes[i+1:], st.fg)
			i += used
			continue
		}
		st.applyCode(codes[i])
	}
}

func (st *style) applyCode(code int) {
	switch code {
	case sgrReset:
		*st = style{}
	case sgrBold:
		st.bold = true
	case sgrDim:
		st.dim = true
	case sgrItalic:
		st.italic = true
	case sgrUnderline:
		st.underline = true
	default:
		if c, ok := basicColor(code); ok {
			st.fg = c
		}
	}
}

func basicColor(code int) (color, bool) {
	switch {
	case code >= sgrFGBase && code <= sgrFGEnd:
		return color{kind: kindIndexed, index: code - sgrFGBase}, true
	case code >= sgrBrightBase && code <= sgrBrightEnd:
		return color{kind: kindIndexed, index: code - sgrBrightBase + brightOffset}, true
	default:
		return color{}, false
	}
}

func parseCodes(params string) []int {
	if params == "" {
		return []int{sgrReset}
	}
	fields := strings.Split(params, ";")
	codes := make([]int, len(fields))
	for i, f := range fields {
		codes[i], _ = strconv.Atoi(f)
	}
	return codes
}

// extendedColor reads the arguments of a 38 sequence and reports how many it consumed.
func extendedColor(args []int, current color) (result color, consumed int) {
	if len(args) >= 2 && args[0] == modeIndexed {
		return color{kind: kindIndexed, index: clamp(args[1])}, 2
	}
	if len(args) >= 4 && args[0] == modeRGB {
		return color{kind: kindRGB, r: clamp(args[1]), g: clamp(args[2]), b: clamp(args[3])}, 4
	}
	return current, len(args)
}

func clamp(n int) int {
	return max(0, min(n, 255))
}

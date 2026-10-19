// Package style reads and writes the style row format the feature files use to describe colored terminal
// output. Each row of text is followed by a line with one style character under every column, and a legend
// names the styles that have no built-in character.
package style

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/layout"
)

const (
	unstyled     = ' '
	continuation = '_'
	wideColumns  = 2
)

var (
	errEmptyRow      = errors.New("a row of an output with styles cannot be empty")
	errTooManyStyles = errors.New("more styles than legend characters")
	errWideRune      = errors.New("bad style line under a wide character")
	errUnknownStyle  = errors.New("unknown style")
	errUnsupported   = errors.New("unsupported output")
	errLegend        = errors.New("bad legend")
)

var colorNames = [...]string{"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"}

var builtins = map[rune]Style{
	'd': {Dim: true},
	'B': {Bold: true},
	'r': {FG: "red"},
	'g': {FG: "green"},
	'y': {FG: "yellow"},
	'b': {FG: "blue"},
	'm': {FG: "magenta"},
	'c': {FG: "cyan"},
	'w': {FG: "white"},
	'k': {FG: "bright-black"},
}

var (
	builtinChars = reverseBuiltins()
	legendChars  = legendAlphabet()
)

// Style is the set of SGR attributes in effect for one cell. FG and BG hold a color name such as "red" or
// "bright-black", "#rrggbb" for a true color or "256:n" for a palette color; empty means the default.
type Style struct {
	Bold bool
	Dim  bool
	FG   string
	BG   string
}

// Cell is one character of output, with any zero-width characters that follow it, and the style it was
// printed in. A cell is Width columns wide.
type Cell struct {
	Text  string
	Width int
	Style Style
}

// Row is one line of output.
type Row []Cell

// Block is the rows of an output.
type Block []Row

func reverseBuiltins() map[Style]rune {
	m := make(map[Style]rune, len(builtins))
	for char, s := range builtins {
		m[s] = char
	}
	return m
}

func legendAlphabet() []rune {
	var chars []rune
	for c := '1'; c <= '9'; c++ {
		chars = append(chars, c)
	}
	for _, r := range "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		if _, ok := builtins[r]; !ok {
			chars = append(chars, r)
		}
	}
	return chars
}

// Text returns the rows with every style removed, one per line.
func (b Block) Text() string {
	lines := make([]string, len(b))
	for i, row := range b {
		lines[i] = row.text()
	}
	return strings.Join(lines, "\n")
}

// Styled reports whether any cell has a style.
func (b Block) Styled() bool {
	for _, row := range b {
		for _, c := range row {
			if c.Style != (Style{}) {
				return true
			}
		}
	}
	return false
}

// Equal reports whether both blocks hold the same text in the same styles.
func (b Block) Equal(other Block) bool {
	if len(b) != len(other) {
		return false
	}
	for i := range b {
		if len(b[i]) != len(other[i]) {
			return false
		}
		for j := range b[i] {
			if b[i][j] != other[i][j] {
				return false
			}
		}
	}
	return true
}

func (r Row) text() string {
	var sb strings.Builder
	for _, c := range r {
		sb.WriteString(c.Text)
	}
	return sb.String()
}

func (r Row) trimmed() Row {
	end := len(r)
	for end > 0 && r[end-1].Text == " " && r[end-1].Style == (Style{}) {
		end--
	}
	return r[:end]
}

func (r Row) appendRune(c rune, s Style) (Row, error) {
	w := layout.RuneWidth(c)
	if w == 0 {
		if len(r) == 0 {
			return nil, fmt.Errorf("%w: the zero-width character %U starts a row", errUnsupported, c)
		}
		r[len(r)-1].Text += string(c)
		return r, nil
	}
	return append(r, Cell{Text: string(c), Width: min(w, wideColumns), Style: s}), nil
}

func (s Style) spec() string {
	var parts []string
	if s.Bold {
		parts = append(parts, "bold")
	}
	if s.Dim {
		parts = append(parts, "dim")
	}
	if s.FG != "" {
		parts = append(parts, s.FG)
	}
	if s.BG != "" {
		parts = append(parts, "bg:"+s.BG)
	}
	return strings.Join(parts, " ")
}

func parseSpec(spec string) (Style, error) {
	var s Style
	for token := range strings.FieldsSeq(spec) {
		if err := s.applyToken(token); err != nil {
			return Style{}, err
		}
	}
	if s == (Style{}) {
		return Style{}, fmt.Errorf("%w: %q has no style", errLegend, spec)
	}
	return s, nil
}

func (s *Style) applyToken(token string) error {
	switch token {
	case "bold":
		s.Bold = true
		return nil
	case "dim":
		s.Dim = true
		return nil
	}
	target, color := &s.FG, token
	if rest, ok := strings.CutPrefix(token, "bg:"); ok {
		target, color = &s.BG, rest
	}
	if !validColor(color) {
		return fmt.Errorf("%w %q", errUnknownStyle, token)
	}
	if *target != "" {
		return fmt.Errorf("%w: two colors of one kind at %q", errLegend, token)
	}
	*target = color
	return nil
}

func validColor(color string) bool {
	if hex, ok := strings.CutPrefix(color, "#"); ok {
		_, err := strconv.ParseUint(hex, 16, 32)
		return len(hex) == 6 && err == nil
	}
	if n, ok := strings.CutPrefix(color, "256:"); ok {
		v, err := strconv.Atoi(n)
		return err == nil && v >= 0 && v <= 255 && n == strconv.Itoa(v)
	}
	name := strings.TrimPrefix(color, "bright-")
	for _, known := range colorNames {
		if name == known {
			return true
		}
	}
	return false
}

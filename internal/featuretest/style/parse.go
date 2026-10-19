package style

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/PaddleHQ/claude-statusline/internal/layout"
)

var legendLine = regexp.MustCompile(`^(\S) = (.+)$`)

// Parse reads a block written in the style row format: each text row is followed by its style line, and a
// blank line where a text row would start begins the legend. A row without styles has an empty style line.
// Trailing unstyled spaces are trimmed from every row.
func Parse(doc string) (Block, error) {
	if doc == "" {
		return nil, nil
	}
	rowLines, legendLines := splitLegend(strings.Split(doc, "\n"))
	legend, err := parseLegend(legendLines)
	if err != nil {
		return nil, err
	}
	used := make(map[rune]bool)
	var block Block
	for i := 0; i < len(rowLines); i += 2 {
		styleLine := ""
		if i+1 < len(rowLines) {
			styleLine = rowLines[i+1]
		}
		row, err := parseRow(rowLines[i], styleLine, legend, used)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i/2+1, err)
		}
		block = append(block, row.trimmed())
	}
	for char := range legend {
		if !used[char] {
			return nil, fmt.Errorf("%w: %q is not used by any row", errLegend, char)
		}
	}
	return block, nil
}

func splitLegend(lines []string) (rows, legend []string) {
	for i := 0; i < len(lines); i += 2 {
		if lines[i] == "" {
			return lines[:i], lines[i+1:]
		}
	}
	return lines, nil
}

func parseLegend(lines []string) (map[rune]Style, error) {
	legend := make(map[rune]Style)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		m := legendLine.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("%w: %q is not in the form \"<character> = <style>\"", errLegend, line)
		}
		char, _ := utf8.DecodeRuneInString(m[1])
		if _, ok := builtins[char]; ok || char == continuation {
			return nil, fmt.Errorf("%w: %q is built in and cannot be redefined", errLegend, char)
		}
		if _, ok := legend[char]; ok {
			return nil, fmt.Errorf("%w: %q is defined twice", errLegend, char)
		}
		s, err := parseSpec(m[2])
		if err != nil {
			return nil, fmt.Errorf("legend %q: %w", char, err)
		}
		legend[char] = s
	}
	return legend, nil
}

type styleLine struct {
	chars  []rune
	legend map[rune]Style
	used   map[rune]bool
}

func (l styleLine) at(col int) rune {
	if col < len(l.chars) {
		return l.chars[col]
	}
	return unstyled
}

func (l styleLine) styleAt(col int) (Style, error) {
	char := l.at(col)
	if char == unstyled {
		return Style{}, nil
	}
	if s, ok := builtins[char]; ok {
		return s, nil
	}
	if s, ok := l.legend[char]; ok {
		l.used[char] = true
		return s, nil
	}
	return Style{}, fmt.Errorf("%w: %q in column %d of the style line", errUnknownStyle, char, col+1)
}

func parseRow(text, styles string, legend map[rune]Style, used map[rune]bool) (Row, error) {
	line := styleLine{chars: []rune(styles), legend: legend, used: used}
	var (
		row Row
		col int
	)
	for _, r := range text {
		w := layout.RuneWidth(r)
		if w == 0 {
			var err error
			if row, err = row.appendRune(r, Style{}); err != nil {
				return nil, err
			}
			continue
		}
		w = min(w, wideColumns)
		s, err := line.styleAt(col)
		if err != nil {
			return nil, err
		}
		if w == wideColumns {
			if err := line.checkContinuation(col, s); err != nil {
				return nil, err
			}
		}
		row = append(row, Cell{Text: string(r), Width: w, Style: s})
		col += w
	}
	for ; col < len(line.chars); col++ {
		s, err := line.styleAt(col)
		if err != nil {
			return nil, err
		}
		row = append(row, Cell{Text: " ", Width: 1, Style: s})
	}
	return row, nil
}

func (l styleLine) checkContinuation(col int, s Style) error {
	want := rune(unstyled)
	if s != (Style{}) {
		want = continuation
	}
	if got := l.at(col + 1); got != want {
		return fmt.Errorf("%w: column %d is %q but must be %q", errWideRune, col+2, got, want)
	}
	return nil
}

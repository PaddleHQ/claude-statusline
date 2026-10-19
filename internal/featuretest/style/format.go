package style

import (
	"fmt"
	"strings"
)

type legendBuilder struct {
	assigned map[Style]rune
	styles   []Style
}

func (l *legendBuilder) char(s Style) (rune, error) {
	if s == (Style{}) {
		return unstyled, nil
	}
	if char, ok := builtinChars[s]; ok {
		return char, nil
	}
	if char, ok := l.assigned[s]; ok {
		return char, nil
	}
	if len(l.styles) >= len(legendChars) {
		return 0, errTooManyStyles
	}
	char := legendChars[len(l.styles)]
	l.assigned[s] = char
	l.styles = append(l.styles, s)
	return char, nil
}

func (l *legendBuilder) lines() []string {
	if len(l.styles) == 0 {
		return nil
	}
	lines := []string{""}
	for i, s := range l.styles {
		lines = append(lines, fmt.Sprintf("%c = %s", legendChars[i], s.spec()))
	}
	return lines
}

// Format writes the block in the style row format, with a legend for the styles that have no built-in
// character. Legend characters are the digits 1 to 9 and then the letters no built-in uses, in the order the
// styles first appear.
func (b Block) Format() (string, error) {
	styled := b.Styled()
	legend := &legendBuilder{assigned: make(map[Style]rune)}
	var lines []string
	for _, row := range b {
		if len(row) == 0 && styled {
			return "", errEmptyRow
		}
		var styles strings.Builder
		for _, c := range row {
			char, err := legend.char(c.Style)
			if err != nil {
				return "", err
			}
			styles.WriteRune(char)
			if c.Width == wideColumns {
				styles.WriteRune(continuationFor(char))
			}
		}
		lines = append(lines, row.text(), strings.TrimRight(styles.String(), " "))
	}
	return strings.Join(append(lines, legend.lines()...), "\n"), nil
}

func continuationFor(char rune) rune {
	if char == unstyled {
		return unstyled
	}
	return continuation
}

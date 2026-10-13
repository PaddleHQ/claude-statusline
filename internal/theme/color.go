package theme

import (
	"fmt"
	"strconv"
	"strings"
)

var colorNames = []string{"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"}

var attributes = map[string]string{
	"bold":      "\033[1m",
	"dim":       "\033[2m",
	"italic":    "\033[3m",
	"underline": "\033[4m",
}

const (
	basicFG     = 30
	brightFG    = 90
	maxIndex    = 255
	hexDigits   = 6
	hexBase     = 16
	channelBits = 8
)

// ParseColor compiles a color spec such as "bold 33" into escape sequences.
// An empty spec means no styling. A spec may hold at most one color.
func ParseColor(spec string) (string, error) {
	var b strings.Builder
	colors := 0
	for _, tok := range strings.Fields(spec) {
		if seq, ok := attributes[tok]; ok {
			b.WriteString(seq)
			continue
		}
		seq, err := colorSequence(tok)
		if err != nil {
			return "", err
		}
		if colors++; colors > 1 {
			return "", fmt.Errorf("color spec %q has more than one color", spec)
		}
		b.WriteString(seq)
	}
	return b.String(), nil
}

func colorSequence(tok string) (string, error) {
	if rgb, ok := strings.CutPrefix(tok, "#"); ok {
		r, g, bl, err := parseHex(rgb)
		if err != nil {
			return "", fmt.Errorf("color %q: %w", tok, err)
		}
		return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, bl), nil
	}
	if isDigits(tok) {
		n, err := strconv.Atoi(tok)
		if err != nil || n > maxIndex {
			return "", fmt.Errorf("color %q is not in 0-%d", tok, maxIndex)
		}
		return fmt.Sprintf("\033[38;5;%dm", n), nil
	}
	base, name := basicFG, tok
	if rest, ok := strings.CutPrefix(tok, "bright-"); ok {
		base, name = brightFG, rest
	}
	for i, n := range colorNames {
		if n == name {
			return fmt.Sprintf("\033[%dm", base+i), nil
		}
	}
	return "", fmt.Errorf("unknown color token %q", tok)
}

func parseHex(s string) (r, g, b int, err error) {
	if len(s) != hexDigits {
		return 0, 0, 0, fmt.Errorf("want 6 hex digits")
	}
	v, err := strconv.ParseUint(s, hexBase, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("want 6 hex digits")
	}
	mask := uint64(1<<channelBits - 1)
	return int(v >> (2 * channelBits) & mask), int(v >> channelBits & mask), int(v & mask), nil
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

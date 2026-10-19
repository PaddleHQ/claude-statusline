package style

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	esc        = '\033'
	extendedFG = 38
	extendedBG = 48
	paletteArg = "5"
	rgbArg     = "2"
	maxChannel = 255
)

// Decode reads terminal output, applying each SGR sequence to the text after it. Any other escape sequence,
// and any SGR parameter it does not know, is an error. Trailing unstyled spaces are trimmed from every row.
func Decode(raw string) (Block, error) {
	if raw == "" {
		return nil, nil
	}
	var (
		block Block
		row   Row
		state Style
	)
	runes := []rune(raw)
	for i := 0; i < len(runes); i++ {
		switch runes[i] {
		case '\n':
			block = append(block, row.trimmed())
			row = nil
		case esc:
			end, next, err := readSGR(runes, i, state)
			if err != nil {
				return nil, err
			}
			state, i = next, end
		default:
			var err error
			if row, err = row.appendRune(runes[i], state); err != nil {
				return nil, err
			}
		}
	}
	return append(block, row.trimmed()), nil
}

func readSGR(runes []rune, start int, state Style) (end int, next Style, err error) {
	if start+1 >= len(runes) || runes[start+1] != '[' {
		return 0, Style{}, fmt.Errorf("%w: an escape that does not start a CSI sequence", errUnsupported)
	}
	for end = start + 2; end < len(runes); end++ {
		if runes[end] >= 0x40 && runes[end] <= 0x7e {
			break
		}
	}
	if end >= len(runes) {
		return 0, Style{}, fmt.Errorf("%w: an unterminated escape sequence", errUnsupported)
	}
	body := string(runes[start+2 : end])
	if runes[end] != 'm' {
		return 0, Style{}, fmt.Errorf("%w: ESC[%s%c is not a color or style sequence", errUnsupported, body, runes[end])
	}
	next, err = applySGR(state, body)
	return end, next, err
}

func applySGR(state Style, body string) (Style, error) {
	params := strings.Split(body, ";")
	for i := 0; i < len(params); i++ {
		code := 0
		if params[i] != "" {
			var err error
			if code, err = strconv.Atoi(params[i]); err != nil {
				return Style{}, fmt.Errorf("%w: SGR parameter %q in ESC[%sm", errUnsupported, params[i], body)
			}
		}
		consumed, ok := state.apply(code, params[i+1:])
		if !ok {
			return Style{}, fmt.Errorf("%w: SGR parameter %d in ESC[%sm", errUnsupported, code, body)
		}
		i += consumed
	}
	return state, nil
}

func (s *Style) apply(code int, rest []string) (consumed int, ok bool) {
	switch code {
	case 0:
		*s = Style{}
	case 1:
		s.Bold = true
	case 2:
		s.Dim = true
	case 22:
		s.Bold, s.Dim = false, false
	case 39:
		s.FG = ""
	case 49:
		s.BG = ""
	case extendedFG:
		return extended(&s.FG, rest)
	case extendedBG:
		return extended(&s.BG, rest)
	default:
		return 0, s.applyColor(code)
	}
	return 0, true
}

func (s *Style) applyColor(code int) bool {
	switch {
	case code >= 30 && code <= 37:
		s.FG = colorNames[code-30]
	case code >= 40 && code <= 47:
		s.BG = colorNames[code-40]
	case code >= 90 && code <= 97:
		s.FG = "bright-" + colorNames[code-90]
	case code >= 100 && code <= 107:
		s.BG = "bright-" + colorNames[code-100]
	default:
		return false
	}
	return true
}

func extended(target *string, rest []string) (consumed int, ok bool) {
	if len(rest) == 0 {
		return 0, false
	}
	switch rest[0] {
	case paletteArg:
		n, ok := channel(rest, 1)
		if !ok {
			return 0, false
		}
		*target = "256:" + strconv.Itoa(n)
		return 2, true
	case rgbArg:
		var rgb [3]int
		for i := range rgb {
			if rgb[i], ok = channel(rest, i+1); !ok {
				return 0, false
			}
		}
		*target = fmt.Sprintf("#%02x%02x%02x", rgb[0], rgb[1], rgb[2])
		return 4, true
	}
	return 0, false
}

func channel(params []string, i int) (int, bool) {
	if i >= len(params) {
		return 0, false
	}
	v, err := strconv.Atoi(params[i])
	return v, err == nil && v >= 0 && v <= maxChannel
}

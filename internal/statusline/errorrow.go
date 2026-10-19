package statusline

import (
	"strconv"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
)

const (
	errorHint     = " · run /statusline:config-check"
	errorEllipsis = "…"
)

// errorText is the row's text while it is being fitted to the terminal.
type errorText struct {
	parts  []string
	keep   int
	line   int
	reason string
	hint   string
}

// errorRow is the single row shown above the line when the config file could not be used.
// A columns of zero means no limit.
func errorRow(p *config.Problem, th *theme.Theme, home string, columns int) string {
	parts := strings.Split(config.Clean(collapseHome(p.Path, home)), "/")
	e := &errorText{parts: parts, keep: len(parts), line: p.Line, reason: config.Clean(p.Reason), hint: errorHint}
	return th.Bad(e.fit(columns))
}

// collapseHome writes the home directory as ~, so the path stays short and does not name the user.
func collapseHome(path, home string) string {
	home = strings.TrimRight(home, "/")
	if home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(path, home+"/"); ok {
		return "~/" + rest
	}
	return path
}

func (e *errorText) path() string {
	if e.keep >= len(e.parts) {
		return strings.Join(e.parts, "/")
	}
	return errorEllipsis + "/" + strings.Join(e.parts[len(e.parts)-e.keep:], "/")
}

func (e *errorText) String() string {
	where := ": "
	if e.line > 0 {
		where = ":" + strconv.Itoa(e.line) + " "
	}
	return e.path() + where + e.reason + e.hint
}

// fit shortens the text until it is at most columns wide: the path first, then the hint, then the reason.
func (e *errorText) fit(columns int) string {
	if columns <= 0 {
		return e.String()
	}
	for e.keep > 1 && layout.Width(e.String()) > columns {
		e.keep--
	}
	if layout.Width(e.String()) > columns {
		e.hint = ""
	}
	e.cutReason(layout.Width(e.String()) - columns)
	return cutRunes(e.String(), columns)
}

// cutReason removes excess columns from the end of the reason, marking the cut with an ellipsis.
func (e *errorText) cutReason(excess int) {
	runes := []rune(e.reason)
	if excess <= 0 || len(runes) == 0 {
		return
	}
	e.reason = string(runes[:max(len(runes)-excess-1, 0)]) + errorEllipsis
}

func cutRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

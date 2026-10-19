// Package layout arranges rendered segments into one row, or wraps them into several when the terminal is too narrow.
package layout

import (
	"regexp"
	"strings"

	"github.com/mattn/go-runewidth"
)

// Group says where a segment goes when the line has to wrap.
type Group int

const (
	// Identity segments (path, model, context) pack greedily into the top rows.
	Identity Group = iota
	// Usage segments (tokens, cache) sit flush left on the usage row.
	Usage
	// Rates segments (the rate-limit bars) stay together and line up under the second identity segment.
	Rates
	// Extra segments take a row of their own.
	Extra
)

// Item is one rendered, non-empty segment.
type Item struct {
	Group Group
	Text  string
}

// Options describes the terminal and how rows are joined.
type Options struct {
	Columns   int
	Separator string
	// Anchor is drawn at the start of an indented row, because Claude Code trims leading spaces but not a glyph.
	Anchor string
}

// minRoom is the narrowest a wrapped row may get before the indent is dropped.
const minRoom = 24

var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

// columns ignores the locale, so wrapping does not change under a CJK locale that would widen ambiguous characters.
var columns = func() *runewidth.Condition {
	c := runewidth.NewCondition()
	c.EastAsianWidth = false
	return c
}()

// RuneWidth is the number of columns r occupies on its own. A rune followed by a variation selector takes more,
// which only Width sees.
func RuneWidth(r rune) int {
	return columns.RuneWidth(r)
}

const (
	variationSelector16 = '️'
	zeroWidthJoiner     = '‍'
)

// Width is the number of columns s occupies once color sequences are removed, counting wide characters as two.
// The feature-test harness measures its cells with it too, so there is one rule.
func Width(s string) int {
	plain := sgr.ReplaceAllString(s, "")
	return columns.StringWidth(plain) + emojiSelectorColumns(plain)
}

// emojiSelectorColumns is the extra columns for each narrow base character that U+FE0F turns into an emoji.
// go-runewidth counts such a pair as one column, but string-width (which Claude Code measures with) and
// terminals that draw the emoji use two.
func emojiSelectorColumns(s string) int {
	runes := []rune(s)
	extra := 0
	for i := 1; i < len(runes); i++ {
		if runes[i] != variationSelector16 || columns.RuneWidth(runes[i-1]) != 1 {
			continue
		}
		// Inside a joined sequence such as a heart on fire, go-runewidth already counts the whole emoji.
		if i+1 < len(runes) && runes[i+1] == zeroWidthJoiner {
			continue
		}
		extra++
	}
	return extra
}

// Render joins the items on one row when they fit within o.Columns, and otherwise wraps them into rows.
// A width of zero means the width is unknown, so nothing wraps.
func (o *Options) Render(items []Item) string {
	var all []string
	for _, it := range items {
		if it.Text != "" {
			all = append(all, it.Text)
		}
	}
	sepw := Width(o.Separator)
	total := sepw * max(len(all)-1, 0)
	for _, s := range all {
		total += Width(s)
	}
	if o.Columns <= 0 || total <= o.Columns {
		return strings.Join(all, o.Separator)
	}
	return wrap(items, *o, sepw)
}

type rows struct {
	o       Options
	sepw    int
	indw    int
	indent  string
	out     strings.Builder
	cur     string
	curw    int
	started bool
}

func wrap(items []Item, o Options, sepw int) string {
	by, firstIdentity := groupItems(items)

	r := newRows(o, sepw, firstIdentity)
	for _, s := range by[Identity] {
		r.emit(s)
	}
	r.flush()

	usage := r.usageRow(strings.Join(by[Usage], o.Separator), strings.Join(by[Rates], o.Separator))
	out := r.out.String() + usage
	if extra := strings.Join(by[Extra], o.Separator); extra != "" {
		if usage != "" {
			out += "\n"
		}
		out += extra
	}
	return strings.TrimSuffix(out, "\n")
}

func groupItems(items []Item) (by map[Group][]string, firstIdentity string) {
	by = map[Group][]string{}
	for _, it := range items {
		if it.Text == "" {
			continue
		}
		if it.Group == Identity && firstIdentity == "" {
			firstIdentity = it.Text
		}
		by[it.Group] = append(by[it.Group], it.Text)
	}
	return by, firstIdentity
}

func newRows(o Options, sepw int, firstIdentity string) *rows {
	r := &rows{o: o, sepw: sepw}
	if firstIdentity != "" {
		r.indw = Width(firstIdentity) + sepw
	}
	if o.Columns-r.indw < minRoom {
		r.indw = 0
	}
	if r.indw > 0 {
		r.indent = o.Anchor + strings.Repeat(" ", r.indw-1)
	}
	return r
}

func (r *rows) usageRow(tok, rates string) string {
	switch {
	case tok != "" && rates != "":
		tokw, ratew := Width(tok), Width(rates)
		end := r.indw + ratew
		if tokw >= r.indw {
			end = tokw + 1 + ratew
		}
		if end <= r.o.Columns {
			return tok + strings.Repeat(" ", max(r.indw-tokw, 1)) + rates
		}
		return tok + "\n" + r.indent + rates
	case tok != "":
		return tok
	case rates != "":
		return r.indent + rates
	}
	return ""
}

func (r *rows) emit(s string) {
	w := Width(s)
	if r.cur == "" {
		r.cur, r.curw = s, w
		return
	}
	limit := r.o.Columns
	if r.started {
		limit -= r.indw
	}
	if r.curw+r.sepw+w <= limit {
		r.cur += r.o.Separator + s
		r.curw += r.sepw + w
		return
	}
	r.write()
	r.cur, r.curw = s, w
}

func (r *rows) flush() {
	if r.cur != "" {
		r.write()
	}
}

func (r *rows) write() {
	if r.started {
		r.out.WriteString(r.indent)
	}
	r.out.WriteString(r.cur + "\n")
	r.started = true
	r.cur, r.curw = "", 0
}

// Package layout arranges rendered segments into one row, or wraps them into several when the terminal is too narrow.
package layout

import (
	"regexp"
	"strings"
	"unicode/utf8"
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

// Width is the number of columns s occupies once color sequences are removed, counting every rune as one column.
func Width(s string) int {
	return utf8.RuneCountInString(sgr.ReplaceAllString(s, ""))
}

// Render joins the items on one row when they fit within o.Columns, and otherwise wraps them into rows.
// A width of zero means the width is unknown, so nothing wraps.
func Render(items []Item, o Options) string {
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
	return wrap(items, o, sepw)
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

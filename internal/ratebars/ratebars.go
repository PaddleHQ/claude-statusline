// Package ratebars prints the 5h and 7d usage bars as JSON, painted in the user's theme, for tools that draw them in
// their own layout.
package ratebars

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments"
	"github.com/PaddleHQ/claude-statusline/segments/rate"
)

const (
	keyTheme      = "theme"
	minCellsLimit = 1
	maxCellsLimit = 200
	absent        = "--"
)

// ErrUsage is returned for flags that are missing, malformed or out of range.
var ErrUsage = errors.New("usage: claude-statusline rate-bars --min-cells=N --max-cells=M (1 <= N <= M <= 200)")

// Window is one usage window: its percentage and a bar for each requested width.
type Window struct {
	OK      bool   `json:"ok"`
	Percent string `json:"percent"`
	Bars    Series `json:"bars"`
}

// Series holds one painted bar per cell count from Min upwards, and encodes as an object keyed by the count in
// ascending order.
type Series struct {
	Min  int
	Bars []string
}

// MarshalJSON encodes the bars keyed by cell count, in numeric order rather than the lexical order of a map.
func (s Series) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, painted := range s.Bars {
		if i > 0 {
			b.WriteByte(',')
		}
		value, err := encode(painted)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&b, "%q:%s", strconv.Itoa(s.Min+i), value)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// Output is the command's JSON object.
type Output struct {
	Rate5h Window `json:"rate5h"`
	Rate7d Window `json:"rate7d"`
}

// Build parses the flags and returns the JSON line. It reads the shared rate-limit cache and never writes it.
func Build(args []string, getenv func(string) string) (string, error) {
	lo, hi, err := parseCells(args)
	if err != nil {
		return "", err
	}

	defaults := segments.Defaults()
	cfg := config.New("", defaults...)
	if path := config.Path(getenv); path != "" {
		cfg = config.Load(path, defaults...)
	}
	th := theme.Load(cfg.String(keyTheme), getenv("HOME"), config.AppDir(getenv))
	if getenv("NO_COLOR") != "" {
		th = th.WithoutColor()
	}

	at := now(getenv)
	var limits ratecache.Limits
	if dir := config.StateDir(getenv); dir != "" {
		cached := ratecache.NewCache(ratecache.PathIn(dir)).Load()
		limits = cached.Visible(at.Unix())
	}

	r := &renderer{th: th, lo: lo, hi: hi}
	out := Output{
		Rate5h: r.window(rate.NewFiveHour(r.deps(cfg.Section("rate5h"), at, &limits))),
		Rate7d: r.window(rate.NewSevenDay(r.deps(cfg.Section("rate7d"), at, &limits))),
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return "", fmt.Errorf("encode rate bars: %w", err)
	}
	return buf.String(), nil
}

func parseCells(args []string) (lo, hi int, err error) {
	fs := flag.NewFlagSet("rate-bars", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.IntVar(&lo, "min-cells", 0, "")
	fs.IntVar(&hi, "max-cells", 0, "")
	if err := fs.Parse(args); err != nil || fs.NArg() > 0 {
		return 0, 0, ErrUsage
	}
	seen := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	if !seen["min-cells"] || !seen["max-cells"] || lo < minCellsLimit || hi > maxCellsLimit || lo > hi {
		return 0, 0, ErrUsage
	}
	return lo, hi, nil
}

func now(getenv func(string) string) time.Time {
	if n, ok := config.ParseNumber(getenv("STATUSLINE_NOW")); ok {
		return time.Unix(int64(n), 0)
	}
	return time.Now()
}

// capture is the Meter the rate segments draw through: it records the pace arguments they derive, so the target and
// thresholds here are the segments' own.
type capture struct{ params bar.Params }

func (c *capture) Pace(usage, target, warn, bad, hardCap int) string {
	c.params = bar.Params{Usage: usage, Target: target, Warn: warn, Bad: bad, HardCap: hardCap}
	return ""
}

type unpainted struct{}

func (unpainted) Text(s string) string { return s }

type renderer struct {
	th     theme.Theme
	lo, hi int
	meter  capture
}

func (r *renderer) deps(cfg *config.Section, at time.Time, limits *ratecache.Limits) rate.Deps {
	return rate.Deps{
		Config: cfg,
		Paint:  unpainted{},
		Meter:  &r.meter,
		Now:    at,
		Limits: func() rate.Limits { return limits },
	}
}

func (r *renderer) window(seg *rate.Window) Window {
	r.meter.params = bar.Params{}
	if seg.Render() == "" {
		return r.absent()
	}
	p := r.meter.params
	bars := make([]string, 0, r.hi-r.lo+1)
	for width := r.lo; width <= r.hi; width++ {
		p.Width = width
		bars = append(bars, p.Render(r.th))
	}
	percent := r.th.Paint(p.PercentColor(r.th), fmt.Sprintf("%d%%", p.Usage))
	return Window{OK: true, Percent: percent, Bars: Series{Min: r.lo, Bars: bars}}
}

func (r *renderer) absent() Window {
	bars := make([]string, 0, r.hi-r.lo+1)
	for width := r.lo; width <= r.hi; width++ {
		empty := bar.Params{Width: width, NoTarget: true}
		bars = append(bars, empty.Render(r.th))
	}
	return Window{OK: false, Percent: r.th.Paint(r.th.Muted, absent), Bars: Series{Min: r.lo, Bars: bars}}
}

func encode(s string) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return "", fmt.Errorf("encode bar: %w", err)
	}
	return string(bytes.TrimSuffix(b.Bytes(), []byte("\n"))), nil
}

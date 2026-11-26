// Package ratebars prints the 5h and 7d usage bars as JSON for tools that draw them in their own layout: painted in the
// user's theme, or with --data as the numbers and pace state for the tool to draw itself.
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
	"github.com/PaddleHQ/claude-statusline/internal/icons"
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
var ErrUsage = errors.New("usage: claude-statusline rate-bars --min-cells=N --max-cells=M (1 <= N <= M <= 200) | --data")

// stateNames names the pace states in the --data output.
var stateNames = map[int]string{theme.StateGood: "good", theme.StateWarn: "warn", theme.StateBad: "bad"}

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
func (s *Series) MarshalJSON() ([]byte, error) {
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

// DataWindow is one usage window as numbers. Without a reading only ok is set: the embedded nil pointer is left out.
type DataWindow struct {
	OK bool `json:"ok"`
	*Reading
}

// Reading is a window's usage, its pace target, both as percentages, the pace state and when the window resets.
type Reading struct {
	Used     int    `json:"used"`
	Target   int    `json:"target"`
	State    string `json:"state"`
	ResetsAt int64  `json:"resets_at"`
}

// DataOutput is the command's JSON object with --data.
type DataOutput struct {
	Rate5h DataWindow `json:"rate5h"`
	Rate7d DataWindow `json:"rate7d"`
}

// Build parses the flags and returns the JSON line. It reads the shared rate-limit cache and never writes it.
func Build(args []string, getenv func(string) string) (string, error) {
	f, err := parseFlags(args)
	if err != nil {
		return "", err
	}

	defaults := segments.Defaults()
	cfg := config.Resolve(getenv, defaults...)
	at := config.Now(getenv)
	var limits ratecache.Limits
	if dir := config.StateDir(getenv); dir != "" {
		cached := ratecache.NewCache(ratecache.PathIn(dir)).Load()
		limits = cached.Visible(at.Unix())
	}

	iconSet := icons.FromConfig(cfg)
	r := &renderer{icons: iconSet, lo: f.lo, hi: f.hi}
	five := rate.NewFiveHour(r.deps(cfg.Section("rate5h"), at, &limits))
	seven := rate.NewSevenDay(r.deps(cfg.Section("rate7d"), at, &limits))
	if f.data {
		return encodeLine(&DataOutput{
			Rate5h: r.data(five, limits.FiveHourReading),
			Rate7d: r.data(seven, limits.SevenDayReading),
		})
	}

	r.th = theme.Load(cfg.String(keyTheme), getenv("HOME"), config.AppDir(getenv))
	if getenv("NO_COLOR") != "" {
		r.th = r.th.WithoutColor()
	}
	return encodeLine(&Output{Rate5h: r.window(five), Rate7d: r.window(seven)})
}

func encodeLine(out any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return "", fmt.Errorf("encode rate bars: %w", err)
	}
	return buf.String(), nil
}

type flags struct {
	lo, hi int
	data   bool
}

// parseFlags accepts either both cell counts or --data alone.
func parseFlags(args []string) (flags, error) {
	var f flags
	fs := flag.NewFlagSet("rate-bars", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.IntVar(&f.lo, "min-cells", 0, "")
	fs.IntVar(&f.hi, "max-cells", 0, "")
	fs.BoolVar(&f.data, "data", false, "")
	if err := fs.Parse(args); err != nil || fs.NArg() > 0 {
		return flags{}, ErrUsage
	}
	seen := map[string]bool{}
	fs.Visit(func(fl *flag.Flag) { seen[fl.Name] = true })
	if f.data && (seen["min-cells"] || seen["max-cells"]) || !f.data && !f.cellsValid(seen) {
		return flags{}, ErrUsage
	}
	return f, nil
}

func (f *flags) cellsValid(seen map[string]bool) bool {
	return seen["min-cells"] && seen["max-cells"] && f.lo >= minCellsLimit && f.hi <= maxCellsLimit && f.lo <= f.hi
}

// capture is the Meter the rate segments draw through: it records the pace arguments they derive, so the target and
// thresholds here are the segments' own.
type capture struct{ params bar.Params }

func (c *capture) Pace(usage, target, warn, bad, hardCap int) string {
	c.params = bar.Params{Usage: usage, Target: target, Warn: warn, Bad: bad, HardCap: hardCap}
	return ""
}

type unpainted struct{}

func (u *unpainted) Text(s string) string { return s }

func (u *unpainted) Dim(s string) string { return s }

type renderer struct {
	th     theme.Theme
	icons  icons.Set
	lo, hi int
	meter  capture
}

func (r *renderer) deps(cfg *config.Section, at time.Time, limits *ratecache.Limits) rate.Deps {
	return rate.Deps{
		Config: cfg,
		Paint:  &unpainted{},
		Meter:  &r.meter,
		Icons:  &r.icons,
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

// data reports the window as numbers, from the same pace arguments the segment draws its bar with.
func (r *renderer) data(seg *rate.Window, reading func() (used, resetsAt int64, ok bool)) DataWindow {
	r.meter.params = bar.Params{}
	_, resetsAt, ok := reading()
	if !ok || seg.Render() == "" {
		return DataWindow{}
	}
	p := r.meter.params
	return DataWindow{OK: true, Reading: &Reading{Used: p.Usage, Target: p.Target, State: stateNames[p.State()], ResetsAt: resetsAt}}
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

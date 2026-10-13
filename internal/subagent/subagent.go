// Package subagent renders the rows of Claude Code's agent panel.
package subagent

import (
	"bytes"
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/segment"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
)

const (
	keyTheme       = "theme"
	keySeparator   = "separator"
	defaultColumns = 999
	percentScale   = 100
	maxPercent     = 1e9
	fallbackName   = "agent"
)

var (
	dateSuffix = regexp.MustCompile(`-[0-9]{8}$`)
	ctxKeys    = []string{"ctx.width", "ctx.width_medium", "ctx.width_narrow", "ctx.warn", "ctx.bad"}
)

// Keys lists every config key the subagent rows read, for checking against defaults.conf.
func Keys() []string {
	return append([]string{keyTheme, keySeparator}, ctxKeys...)
}

type row struct {
	ID      json.RawMessage `json:"id"`
	Content string          `json:"content"`
}

// Build returns one JSON line per task, in the form Claude Code expects, or nothing when the input has no tasks.
func Build(input []byte, getenv func(string) string) string {
	if !gjson.ValidBytes(input) {
		return ""
	}
	root := gjson.ParseBytes(input)
	tasks := root.Get("tasks")
	if !tasks.IsArray() {
		return ""
	}

	cfg := config.New("")
	if path := config.Path(getenv); path != "" {
		cfg = config.Load(path)
	}
	th := theme.Load(cfg.String(keyTheme), getenv("HOME"), config.AppDir(getenv))
	if getenv("NO_COLOR") != "" {
		th = th.WithoutColor()
	}
	columns := defaultColumns
	if c := root.Get("columns"); c.Type == gjson.Number && c.Num >= 0 && c.Num < maxPercent {
		columns = int(c.Num)
	}
	r := renderer{cfg: cfg, th: th, columns: columns}

	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	tasks.ForEach(func(_, task gjson.Result) bool {
		id := task.Get("id")
		if !id.Exists() || id.Type == gjson.Null {
			return true
		}
		_ = enc.Encode(row{ID: json.RawMessage(id.Raw), Content: r.content(task)})
		return true
	})
	return out.String()
}

type renderer struct {
	cfg     *config.Config
	th      theme.Theme
	columns int
}

func (r renderer) content(task gjson.Result) string {
	th := r.th
	name := scalar(task.Get("name"))
	if name == nil {
		name = scalar(task.Get("type"))
	}
	if name == nil {
		fallback := fallbackName
		name = &fallback
	}
	parts := []string{th.Paint(th.Marker, *name)}

	if model := task.Get("model"); truthy(model) && model.Type == gjson.String {
		part := th.Paint(th.Text, shortModel(model.String()))
		if effort := scalar(task.Get("effort")); effort != nil {
			part += th.Paint(th.Muted, " ["+*effort+"]")
		}
		parts = append(parts, part)
	}

	tokens, hasTokens := number(task.Get("tokenCount"))
	if window, ok := number(task.Get("contextWindowSize")); hasTokens && ok && window > 0 {
		parts = append(parts, r.ctx(tokens*percentScale/window))
	}
	if hasTokens {
		parts = append(parts, segment.FormatTokens(int64(math.Min(tokens, maxPercent*maxPercent))))
	}
	if desc := task.Get("description"); desc.Type == gjson.String && desc.String() != "" {
		if text := payload.Sanitize(desc.String()); text != "" {
			parts = append(parts, th.Paint(th.Muted, text))
		}
	}

	sep := " " + th.Paint(th.Muted, r.cfg.String(keySeparator)) + " "
	return strings.Join(parts, sep)
}

func (r renderer) ctx(raw float64) string {
	pct := int(math.Floor(math.Min(math.Max(raw, 0), maxPercent)))
	warn, bad := r.cfg.Int("ctx.warn"), r.cfg.Int("ctx.bad")
	width := bar.WidthFor(r.columns, r.cfg.Int("ctx.width"), r.cfg.Int("ctx.width_medium"), r.cfg.Int("ctx.width_narrow"))
	params := bar.Params{Usage: pct, Width: width, Warn: warn, Bad: bad, NoTarget: true}
	b := bar.Render(params, r.th)
	return r.th.Paint(r.th.Text, "ctx: ") + b + " " + r.th.Paint(bar.Fill(params, r.th), strconv.Itoa(pct)+"%")
}

// truthy mirrors jq: only null, false and a missing field count as absent.
func truthy(r gjson.Result) bool {
	return r.Exists() && r.Type != gjson.Null && r.Type != gjson.False
}

// scalar returns a sanitised string for a present string, number or true, and nil otherwise.
func scalar(r gjson.Result) *string {
	if !truthy(r) {
		return nil
	}
	switch r.Type {
	case gjson.String, gjson.Number, gjson.True:
		s := payload.Sanitize(r.String())
		return &s
	}
	return nil
}

func number(r gjson.Result) (float64, bool) {
	if r.Type != gjson.Number {
		return 0, false
	}
	return r.Num, true
}

// shortModel turns claude-haiku-4-5-20251001 into haiku-4-5.
func shortModel(id string) string {
	id = strings.TrimPrefix(id, "claude-")
	id = dateSuffix.ReplaceAllString(id, "")
	return payload.Sanitize(id)
}

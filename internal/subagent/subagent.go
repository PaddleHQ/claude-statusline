// Package subagent renders the rows of Claude Code's agent panel.
package subagent

import (
	"bytes"
	"encoding/json"
	"math"
	"regexp"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/bar"
	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments"
	"github.com/PaddleHQ/claude-statusline/segments/ctx"
	"github.com/PaddleHQ/claude-statusline/segments/tokens"

	"github.com/tidwall/gjson"
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
	dateSuffix = regexp.MustCompile(`-\d{8}$`)
	// A modern id is claude-<family words>-<major>[-<minor>...], so new models need no table entry. The model segment keeps its own
	// family table because it only needs to tell families apart, not to name them.
	modelID = regexp.MustCompile(`^claude-([a-z]+(?:-[a-z]+)*)-(\d+(?:-\d+)*)$`)
	ctxKeys = []string{"ctx.width", "ctx.width_medium", "ctx.width_narrow", "ctx.warn", "ctx.bad"}
)

// Keys lists every config key the subagent rows read, for checking against defaults.yaml.
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

	defaults := segments.Defaults()
	cfg := config.New("", defaults...)
	if path := config.Path(getenv); path != "" {
		cfg = config.Load(path, defaults...)
	}
	th := theme.Load(cfg.String(keyTheme), getenv("HOME"), config.AppDir(getenv))
	if getenv("NO_COLOR") != "" {
		th = th.WithoutColor()
	}
	columns := defaultColumns
	if c := root.Get("columns"); c.Type == gjson.Number && c.Num >= 0 && c.Num < maxPercent {
		columns = int(c.Num)
	}
	r := &renderer{cfg: cfg, th: th, columns: columns}

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

// reported is a context usage that is already known.
type reported struct{ pct float64 }

func (r *reported) UsedPercentage() (float64, bool) { return r.pct, true }

func (r *renderer) content(task gjson.Result) string {
	th := r.th
	parts := []string{th.Paint(th.Marker, taskName(task))}
	parts = r.appendModel(parts, task)
	parts = r.appendTokens(parts, task)
	parts = r.appendDescription(parts, task)

	sep := " " + th.Paint(th.Muted, r.cfg.String(keySeparator)) + " "
	return strings.Join(parts, sep)
}

func taskName(task gjson.Result) string {
	if name := scalar(task.Get("name")); name != nil {
		return *name
	}
	if name := scalar(task.Get("type")); name != nil {
		return *name
	}
	return fallbackName
}

func (r *renderer) appendModel(parts []string, task gjson.Result) []string {
	model := task.Get("model")
	if !truthy(model) || model.Type != gjson.String {
		return parts
	}
	th := r.th
	part := th.Paint(th.TextColor, modelLabel(model.String()))
	if effort := scalar(task.Get("effort")); effort != nil {
		part += th.Paint(th.Muted, " ["+*effort+"]")
	}
	return append(parts, part)
}

func (r *renderer) appendTokens(parts []string, task gjson.Result) []string {
	count, hasTokens := number(task.Get("tokenCount"))
	if !hasTokens {
		return parts
	}
	if window, ok := number(task.Get("contextWindowSize")); ok && window > 0 {
		parts = append(parts, r.ctx(count*percentScale/window))
	}
	return append(parts, tokens.FormatTokens(int64(math.Min(count, maxPercent*maxPercent))))
}

func (r *renderer) appendDescription(parts []string, task gjson.Result) []string {
	desc := task.Get("description")
	if desc.Type != gjson.String || desc.String() == "" {
		return parts
	}
	if text := payload.Sanitize(desc.String()); text != "" {
		parts = append(parts, r.th.Paint(r.th.Muted, text))
	}
	return parts
}

func (r *renderer) ctx(raw float64) string {
	cfg := r.cfg.Section("ctx")
	meter := &bar.Meter{Theme: r.th, Width: bar.WidthOf(r.columns, cfg), FillPercent: true}
	pct := math.Floor(math.Min(math.Max(raw, 0), maxPercent))
	return ctx.New(ctx.Deps{Session: &reported{pct: pct}, Config: cfg, Paint: &r.th, Meter: meter}).Render()
}

// truthy mirrors jq: only null, false and a missing field count as absent.
func truthy(r gjson.Result) bool {
	return r.Exists() && r.Type != gjson.Null && r.Type != gjson.False
}

// scalar returns a sanitized string for a present string, number or true, and nil otherwise.
func scalar(r gjson.Result) *string {
	if !truthy(r) {
		return nil
	}
	switch r.Type {
	case gjson.String, gjson.Number, gjson.True:
		s := payload.Sanitize(r.String())
		return &s
	default:
		return nil
	}
}

func number(r gjson.Result) (float64, bool) {
	if r.Type != gjson.Number {
		return 0, false
	}
	return r.Num, true
}

// modelLabel turns claude-opus-4-7-20251001 into Opus 4.7, and shows an id it cannot parse as it is minus the claude- prefix.
func modelLabel(id string) string {
	id = payload.Sanitize(id)
	m := modelID.FindStringSubmatch(dateSuffix.ReplaceAllString(id, ""))
	if m == nil {
		return strings.TrimPrefix(id, "claude-")
	}
	words := strings.Split(m[1], "-")
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ") + " " + strings.ReplaceAll(m[2], "-", ".")
}

// Package payload decodes the session JSON that Claude Code writes to stdin.
package payload

import (
	"strings"

	"github.com/tidwall/gjson"
)

// maxCount bounds token counts so sums and percentages cannot overflow.
const maxCount = 1e15

// maxCost bounds the session cost so a bogus value cannot print as a figure wider than the terminal.
const maxCost = 1e9

// Payload holds the fields the status line reads. Strings are safe to print.
type Payload struct {
	Model      Model
	Effort     Effort
	RateLimits RateLimits
	Cwd        string
	// SessionID is empty unless the payload's id is made only of letters, digits, hyphens and underscores.
	SessionID      string
	TranscriptPath string
	ContextWindow  ContextWindow
	// CostUSD is the session's total cost in US dollars; nil means the payload did not report it.
	CostUSD *float64
}

// ContextWindow is the session's context and token usage; a nil field means the payload did not report it.
type ContextWindow struct {
	UsedPercentage *float64
	TotalInput     *int64
	TotalOutput    *int64
	Current        CurrentUsage
}

// CurrentUsage is the token usage of the latest request; absent counts are zero.
type CurrentUsage struct {
	Input         int64
	CacheCreation int64
	CacheRead     int64
}

// Model identifies the model driving the session.
type Model struct {
	DisplayName string
	ID          string
}

// Effort is the session's reasoning effort.
type Effort struct {
	Level string
}

// RateLimits holds the account's usage windows.
type RateLimits struct {
	FiveHour Window
	SevenDay Window
}

// Window is one rate-limit window; a nil field means the payload did not report it.
type Window struct {
	UsedPercentage *float64
	ResetsAt       *float64
}

// Parse never fails: invalid or empty input yields the zero Payload.
func Parse(data []byte) Payload {
	if !gjson.ValidBytes(data) {
		return Payload{}
	}
	root := gjson.ParseBytes(data)
	return Payload{
		Model: Model{
			DisplayName: text(root.Get("model.display_name")),
			ID:          text(root.Get("model.id")),
		},
		Effort: Effort{Level: text(root.Get("effort.level"))},
		RateLimits: RateLimits{
			FiveHour: window(root.Get("rate_limits.five_hour")),
			SevenDay: window(root.Get("rate_limits.seven_day")),
		},
		Cwd:            cwd(root),
		SessionID:      sessionID(root),
		TranscriptPath: text(root.Get("transcript_path")),
		ContextWindow: ContextWindow{
			UsedPercentage: number(root.Get("context_window.used_percentage")),
			TotalInput:     count(root.Get("context_window.total_input_tokens")),
			TotalOutput:    count(root.Get("context_window.total_output_tokens")),
			Current: CurrentUsage{
				Input:         deref(count(root.Get("context_window.current_usage.input_tokens"))),
				CacheCreation: deref(count(root.Get("context_window.current_usage.cache_creation_input_tokens"))),
				CacheRead:     deref(count(root.Get("context_window.current_usage.cache_read_input_tokens"))),
			},
		},
		CostUSD: cost(root.Get("cost.total_cost_usd")),
	}
}

// DisplayName is the model's display name.
func (p *Payload) DisplayName() string { return p.Model.DisplayName }

// ModelID is the model's identifier.
func (p *Payload) ModelID() string { return p.Model.ID }

// EffortLevel is the session's reasoning effort.
func (p *Payload) EffortLevel() string { return p.Effort.Level }

// TranscriptFile is the path of the session transcript.
func (p *Payload) TranscriptFile() string { return p.TranscriptPath }

// WorkingDir is the session's working directory.
func (p *Payload) WorkingDir() string { return p.Cwd }

// LatestRequest is the token usage of the latest request.
func (p *Payload) LatestRequest() (input, cacheCreation, cacheRead int64) {
	u := p.ContextWindow.Current
	return u.Input, u.CacheCreation, u.CacheRead
}

// TotalInput is the session's total input tokens; false means the payload did not report it.
func (p *Payload) TotalInput() (int64, bool) { return reported(p.ContextWindow.TotalInput) }

// TotalOutput is the session's total output tokens; false means the payload did not report it.
func (p *Payload) TotalOutput() (int64, bool) { return reported(p.ContextWindow.TotalOutput) }

// UsedPercentage is how full the context window is; false means the payload did not report it.
func (p *Payload) UsedPercentage() (float64, bool) { return reported(p.ContextWindow.UsedPercentage) }

// TotalCost is the session's total cost in US dollars; false means the payload did not report it.
func (p *Payload) TotalCost() (float64, bool) { return reported(p.CostUSD) }

func reported[T any](v *T) (T, bool) {
	if v == nil {
		var zero T
		return zero, false
	}
	return *v, true
}

func sessionID(root gjson.Result) string {
	id := root.Get("session_id")
	if id.Type != gjson.String || id.String() == "" || strings.ContainsFunc(id.String(), unsafeInPath) {
		return ""
	}
	return id.String()
}

func unsafeInPath(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		return false
	}
	return true
}

func cwd(root gjson.Result) string {
	if dir := text(root.Get("workspace.current_dir")); dir != "" {
		return dir
	}
	return text(root.Get("cwd"))
}

// count reads a token count, rejecting anything too large to add and multiply safely.
func count(r gjson.Result) *int64 {
	n := number(r)
	if n == nil || *n > maxCount {
		return nil
	}
	c := int64(*n)
	return &c
}

func cost(r gjson.Result) *float64 {
	n := number(r)
	if n == nil || *n > maxCost {
		return nil
	}
	return n
}

func deref(n *int64) int64 {
	if n == nil {
		return 0
	}
	return *n
}

func window(r gjson.Result) Window {
	return Window{
		UsedPercentage: number(r.Get("used_percentage")),
		ResetsAt:       number(r.Get("resets_at")),
	}
}

func number(r gjson.Result) *float64 {
	if r.Type != gjson.Number || r.Num < 0 {
		return nil
	}
	n := r.Num
	return &n
}

// text accepts scalars only, so a nested object never reaches the terminal as raw JSON.
func text(r gjson.Result) string {
	switch r.Type {
	case gjson.String, gjson.Number, gjson.True, gjson.False:
		return Sanitize(r.String())
	default:
		return ""
	}
}

// Sanitize drops control characters so a payload cannot inject terminal escapes.
func Sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return -1
		}
		return r
	}, s)
}

// Package payload decodes the session JSON that Claude Code writes to stdin.
package payload

import (
	"strings"

	"github.com/tidwall/gjson"
)

// maxCount bounds token counts so sums and percentages cannot overflow.
const maxCount = 1e15

// Payload holds the fields the status line reads. Strings are safe to print.
type Payload struct {
	Model         Model
	Effort        Effort
	RateLimits    RateLimits
	Cwd           string
	ContextWindow ContextWindow
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
		Cwd: cwd(root),
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
	}
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
	}
	return ""
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

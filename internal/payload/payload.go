// Package payload decodes the session JSON that Claude Code writes to stdin.
package payload

import (
	"strings"

	"github.com/tidwall/gjson"
)

// Payload holds the fields the status line reads. Strings are safe to print.
type Payload struct {
	Model      Model
	Effort     Effort
	RateLimits RateLimits
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
	}
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
		return sanitize(r.String())
	}
	return ""
}

// sanitize drops control characters so a payload cannot inject terminal escapes.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return -1
		}
		return r
	}, s)
}

package payload

import (
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("reads model and effort", func(t *testing.T) {
		t.Parallel()
		p := Parse([]byte(`{"model":{"id":"claude-opus-4-7","display_name":"Opus 4.7"},"effort":{"level":"high"}}`))
		if p.Model.DisplayName != "Opus 4.7" || p.Model.ID != "claude-opus-4-7" || p.Effort.Level != "high" {
			t.Errorf("Parse() = %+v", p)
		}
	})

	t.Run("reads rate limits", func(t *testing.T) {
		t.Parallel()
		p := Parse([]byte(`{"rate_limits":{"five_hour":{"used_percentage":45.4,"resets_at":1760009000},"seven_day":{"used_percentage":40}}}`))
		if p.RateLimits.FiveHour.UsedPercentage == nil || *p.RateLimits.FiveHour.UsedPercentage != 45.4 {
			t.Errorf("five-hour used = %v", p.RateLimits.FiveHour.UsedPercentage)
		}
		if p.RateLimits.FiveHour.ResetsAt == nil || *p.RateLimits.FiveHour.ResetsAt != 1760009000 {
			t.Errorf("five-hour reset = %v", p.RateLimits.FiveHour.ResetsAt)
		}
		if p.RateLimits.SevenDay.UsedPercentage == nil || *p.RateLimits.SevenDay.UsedPercentage != 40 {
			t.Errorf("seven-day used = %v", p.RateLimits.SevenDay.UsedPercentage)
		}
		if p.RateLimits.SevenDay.ResetsAt != nil {
			t.Errorf("seven-day reset = %v, want nil", *p.RateLimits.SevenDay.ResetsAt)
		}
	})
}

func TestParseRejectsBadRateValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"negative", `{"rate_limits":{"five_hour":{"used_percentage":-1}}}`},
		{"string", `{"rate_limits":{"five_hour":{"used_percentage":"45"}}}`},
		{"null", `{"rate_limits":{"five_hour":{"used_percentage":null}}}`},
		{"missing", `{"rate_limits":{}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Parse([]byte(tt.input)).RateLimits.FiveHour.UsedPercentage; got != nil {
				t.Errorf("used percentage = %v, want nil", *got)
			}
		})
	}
}

func TestParseText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", `{"model":{"display_name":"Sonnet"}}`, "Sonnet"},
		{"strips control characters", "{\"model\":{\"display_name\":\"Op\\u001b[31mus\\u0000\\n\\u007f 4\"}}", "Op[31mus 4"},
		{"strips C1 control characters", "{\"model\":{\"display_name\":\"Op\\u009b31mus\\u0080 4\\u009f\"}}", "Op31mus 4"},
		{"keeps non-ASCII", `{"model":{"display_name":"Opus ·é"}}`, "Opus ·é"},
		{"stringifies numbers", `{"model":{"display_name":5}}`, "5"},
		{"ignores objects", `{"model":{"display_name":{"a":"b"}}}`, ""},
		{"ignores arrays", `{"model":{"display_name":["a"]}}`, ""},
		{"ignores null", `{"model":{"display_name":null}}`, ""},
		{"missing model", `{"effort":{"level":"high"}}`, ""},
		{"model of the wrong type", `{"model":"Opus"}`, ""},
		{"ignores unknown fields", `{"extra":[1,2],"model":{"display_name":"Sonnet","other":true}}`, "Sonnet"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Parse([]byte(tt.input)).Model.DisplayName; got != tt.want {
				t.Errorf("display name = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseSanitizesEveryString(t *testing.T) {
	t.Parallel()

	p := Parse([]byte("{\"model\":{\"id\":\"a\\u001bb\"},\"effort\":{\"level\":\"h\\u0007i\"}}"))
	if p.Model.ID != "ab" || p.Effort.Level != "hi" {
		t.Errorf("Parse() = %+v", p)
	}
}

func TestParseInvalidInput(t *testing.T) {
	t.Parallel()

	for _, input := range []string{``, `{"model":`, `not json`, `[]`, `null`, `{"model":{"display_name":"x"}} trailing`} {
		if got := Parse([]byte(input)); got != (Payload{}) {
			t.Errorf("Parse(%q) = %+v, want zero", input, got)
		}
	}
}

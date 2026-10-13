package payload_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/PaddleHQ/claude-statusline/internal/payload"
)

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("reads model and effort", func(t *testing.T) {
		t.Parallel()
		p := payload.Parse([]byte(`{"model":{"id":"claude-opus-4-7","display_name":"Opus 4.7"},"effort":{"level":"high"}}`))
		assert.Equal(t, "Opus 4.7", p.Model.DisplayName)
		assert.Equal(t, "claude-opus-4-7", p.Model.ID)
		assert.Equal(t, "high", p.Effort.Level)
	})

	t.Run("reads rate limits", func(t *testing.T) {
		t.Parallel()
		p := payload.Parse([]byte(`{"rate_limits":{"five_hour":{"used_percentage":45.4,"resets_at":1760009000},"seven_day":{"used_percentage":40}}}`))
		require.NotNil(t, p.RateLimits.FiveHour.UsedPercentage)
		assert.Equal(t, 45.4, *p.RateLimits.FiveHour.UsedPercentage)
		require.NotNil(t, p.RateLimits.FiveHour.ResetsAt)
		assert.Equal(t, 1760009000.0, *p.RateLimits.FiveHour.ResetsAt)
		require.NotNil(t, p.RateLimits.SevenDay.UsedPercentage)
		assert.Equal(t, 40.0, *p.RateLimits.SevenDay.UsedPercentage)
		assert.Nil(t, p.RateLimits.SevenDay.ResetsAt)
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
			assert.Nil(t, payload.Parse([]byte(tt.input)).RateLimits.FiveHour.UsedPercentage)
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
			assert.Equal(t, tt.want, payload.Parse([]byte(tt.input)).Model.DisplayName)
		})
	}
}

func TestParseSanitizesEveryString(t *testing.T) {
	t.Parallel()

	p := payload.Parse([]byte("{\"model\":{\"id\":\"a\\u001bb\"},\"effort\":{\"level\":\"h\\u0007i\"}}"))
	assert.Equal(t, "ab", p.Model.ID)
	assert.Equal(t, "hi", p.Effort.Level)
}

func TestParseInvalidInput(t *testing.T) {
	t.Parallel()

	for _, input := range []string{``, `{"model":`, `not json`, `[]`, `null`, `{"model":{"display_name":"x"}} trailing`} {
		assert.Equal(t, payload.Payload{}, payload.Parse([]byte(input)), "input %q", input)
	}
}

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

func TestParseWorkingDirectory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"workspace current_dir", `{"workspace":{"current_dir":"/a/b"},"cwd":"/c"}`, "/a/b"},
		{"falls back to cwd", `{"cwd":"/c"}`, "/c"},
		{"empty current_dir falls back", `{"workspace":{"current_dir":""},"cwd":"/c"}`, "/c"},
		{"strips control characters", "{\"cwd\":\"/a\\u001b[31m/b\"}", "/a[31m/b"},
		{"ignores objects", `{"cwd":{"a":"b"}}`, ""},
		{"missing", `{}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, payload.Parse([]byte(tt.input)).Cwd)
		})
	}
}

func TestParseContextWindow(t *testing.T) {
	t.Parallel()

	t.Run("reads every field", func(t *testing.T) {
		t.Parallel()
		p := payload.Parse([]byte(`{"context_window":{"used_percentage":42.5,"total_input_tokens":1500,"total_output_tokens":20,"current_usage":{"input_tokens":1,"cache_creation_input_tokens":2,"cache_read_input_tokens":3}}}`))
		cw := p.ContextWindow
		require.NotNil(t, cw.UsedPercentage)
		assert.Equal(t, 42.5, *cw.UsedPercentage)
		require.NotNil(t, cw.TotalInput)
		assert.Equal(t, int64(1500), *cw.TotalInput)
		require.NotNil(t, cw.TotalOutput)
		assert.Equal(t, int64(20), *cw.TotalOutput)
		assert.Equal(t, payload.CurrentUsage{Input: 1, CacheCreation: 2, CacheRead: 3}, cw.Current)
	})

	t.Run("absent fields stay nil or zero", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, payload.ContextWindow{}, payload.Parse([]byte(`{"context_window":{}}`)).ContextWindow)
	})

	t.Run("a reported zero is kept", func(t *testing.T) {
		t.Parallel()
		cw := payload.Parse([]byte(`{"context_window":{"total_output_tokens":0}}`)).ContextWindow
		require.NotNil(t, cw.TotalOutput)
		assert.Equal(t, int64(0), *cw.TotalOutput)
		assert.Nil(t, cw.TotalInput)
	})

	for _, tt := range []struct{ name, input string }{
		{"negative", `{"context_window":{"total_input_tokens":-5}}`},
		{"string", `{"context_window":{"total_input_tokens":"5"}}`},
		{"null", `{"context_window":{"total_input_tokens":null}}`},
		{"absurdly large", `{"context_window":{"total_input_tokens":1e300}}`},
	} {
		t.Run("rejects "+tt.name+" counts", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, payload.Parse([]byte(tt.input)).ContextWindow.TotalInput)
		})
	}

	t.Run("rejects absurd cache counts", func(t *testing.T) {
		t.Parallel()
		p := payload.Parse([]byte(`{"context_window":{"current_usage":{"cache_read_input_tokens":1e300}}}`))
		assert.Equal(t, payload.CurrentUsage{}, p.ContextWindow.Current)
	})
}

func TestSanitize(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "a[31mb", payload.Sanitize("a\x1b[31mb\x00\n"))
	assert.Equal(t, "é·", payload.Sanitize("é·"))
}

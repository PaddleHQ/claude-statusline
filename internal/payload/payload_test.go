package payload_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/payload"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		assert.InDelta(t, 45.4, *p.RateLimits.FiveHour.UsedPercentage, 1e-9)
		require.NotNil(t, p.RateLimits.FiveHour.ResetsAt)
		assert.InDelta(t, 1760009000.0, *p.RateLimits.FiveHour.ResetsAt, 1e-9)
		require.NotNil(t, p.RateLimits.SevenDay.UsedPercentage)
		assert.InDelta(t, 40.0, *p.RateLimits.SevenDay.UsedPercentage, 1e-9)
		assert.Nil(t, p.RateLimits.SevenDay.ResetsAt)
	})
}

func TestParseTranscriptPath(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/tmp/a.jsonl", payload.Parse([]byte(`{"transcript_path":"/tmp/a.jsonl"}`)).TranscriptPath)
	assert.Empty(t, payload.Parse([]byte(`{"transcript_path":{"a":1}}`)).TranscriptPath)
	assert.Empty(t, payload.Parse([]byte(`{}`)).TranscriptPath)
}

func TestParseSessionID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", `{"session_id":"abc-123_DEF"}`, "abc-123_DEF"},
		{"missing", `{}`, ""},
		{"path traversal", `{"session_id":"../../etc"}`, ""},
		{"separator", `{"session_id":"a/b"}`, ""},
		{"dot", `{"session_id":"."}`, ""},
		{"control character", "{\"session_id\":\"a\\u001bb\"}", ""},
		{"not a string", `{"session_id":123}`, ""},
		{"empty", `{"session_id":""}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, payload.Parse([]byte(tt.input)).SessionID)
		})
	}
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
		assert.InDelta(t, 42.5, *cw.UsedPercentage, 1e-9)
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

func TestSessionAccessors(t *testing.T) {
	t.Parallel()

	p := payload.Parse([]byte(`{"model":{"id":"m","display_name":"M"},"effort":{"level":"high"},"cwd":"/w","transcript_path":"/t",` +
		`"context_window":{"used_percentage":42,"total_input_tokens":7,"total_output_tokens":0,` +
		`"current_usage":{"input_tokens":1,"cache_creation_input_tokens":2,"cache_read_input_tokens":3}},"cost":{"total_cost_usd":1.25}}`))
	assert.Equal(t, "M", p.DisplayName())
	assert.Equal(t, "m", p.ModelID())
	assert.Equal(t, "high", p.EffortLevel())
	assert.Equal(t, "/w", p.WorkingDir())
	assert.Equal(t, "/t", p.TranscriptFile())
	input, creation, read := p.LatestRequest()
	assert.Equal(t, []int64{1, 2, 3}, []int64{input, creation, read})
	pct, ok := p.UsedPercentage()
	assert.True(t, ok)
	assert.InDelta(t, 42.0, pct, 0)
	in, ok := p.TotalInput()
	assert.True(t, ok)
	assert.Equal(t, int64(7), in)
	out, ok := p.TotalOutput()
	assert.True(t, ok)
	assert.Zero(t, out)
	usd, ok := p.TotalCost()
	assert.True(t, ok)
	assert.InDelta(t, 1.25, usd, 0)
}

func TestSessionAccessorsReportWhatIsMissing(t *testing.T) {
	t.Parallel()

	p := payload.Parse([]byte(`{}`))
	_, ok := p.UsedPercentage()
	assert.False(t, ok)
	_, ok = p.TotalInput()
	assert.False(t, ok)
	_, ok = p.TotalOutput()
	assert.False(t, ok)
	_, ok = p.TotalCost()
	assert.False(t, ok)
}

func TestParseCost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  *float64
	}{
		{"reported", `{"cost":{"total_cost_usd":0.42}}`, new(0.42)},
		{"zero is reported", `{"cost":{"total_cost_usd":0}}`, new(0.0)},
		{"at the bound", `{"cost":{"total_cost_usd":1e9}}`, new(1e9)},
		{"above the bound", `{"cost":{"total_cost_usd":1.1e9}}`, nil},
		{"negative", `{"cost":{"total_cost_usd":-1}}`, nil},
		{"a string", `{"cost":{"total_cost_usd":"1.00"}}`, nil},
		{"missing", `{"cost":{}}`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := payload.Parse([]byte(tt.input)).CostUSD
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.InDelta(t, *tt.want, *got, 0)
		})
	}
}

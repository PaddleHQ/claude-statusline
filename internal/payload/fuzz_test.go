package payload_test

import (
	"math"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/payload"

	"github.com/stretchr/testify/assert"
)

func hasControl(s string) bool {
	for _, r := range s {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return true
		}
	}
	return false
}

func FuzzParse(f *testing.F) {
	f.Add([]byte(`{"model":{"id":"claude-opus-4-7","display_name":"Opus 4.7"},"effort":{"level":"high"}}`))
	f.Add([]byte(`{"rate_limits":{"five_hour":{"used_percentage":45.4,"resets_at":1760009000},"seven_day":{"used_percentage":40}}}`))
	f.Add([]byte(`{"session_id":"abc-123_DEF","transcript_path":"/tmp/t.jsonl","workspace":{"current_dir":"/work/repo"},"cwd":"/work"}`))
	f.Add([]byte(`{"context_window":{"used_percentage":12.5,"total_input_tokens":1500,"total_output_tokens":300,"current_usage":{"input_tokens":6,"cache_creation_input_tokens":20000,"cache_read_input_tokens":44500}}}`))
	f.Add([]byte("{\"model\":{\"display_name\":\"Op\\u001b[2Jus\\u009b\\u0000\"},\"cwd\":\"a\\nb\\u007f\"}"))
	f.Add([]byte(`{"model":{"display_name":{"nested":"object"}},"session_id":"../etc"}`))
	f.Add([]byte(`{"cost":{"total_cost_usd":-0}}`))
	f.Add([]byte(``))
	f.Add([]byte(`not json`))

	f.Fuzz(func(t *testing.T, data []byte) {
		p := payload.Parse(data)
		for name, s := range map[string]string{
			"model.display_name": p.Model.DisplayName,
			"model.id":           p.Model.ID,
			"effort.level":       p.Effort.Level,
			"cwd":                p.Cwd,
			"session_id":         p.SessionID,
			"transcript_path":    p.TranscriptPath,
		} {
			assert.False(t, hasControl(s), "%s holds a control character: %q", name, s)
		}
		if usd, ok := p.TotalCost(); ok {
			assert.True(t, usd >= 0 && usd <= 1e9 && !math.Signbit(usd), "cost.total_cost_usd out of range: %v", usd)
		}
	})
}

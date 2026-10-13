package segment_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/segment"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/internal/transcript"
)

func TestDiagnostics(t *testing.T) {
	t.Parallel()

	th := theme.Theme{Text: "<t>", Bad: "<b>", Warn: "<w>", Reset: "</>"}
	tests := []struct {
		name string
		seg  string
		info transcript.Info
		want string
	}{
		{"boot", "boot", transcript.Info{Boot: 64506}, "<t>⌂ 64.5k</>"},
		{"boot hidden at zero", "boot", transcript.Info{}, ""},
		{"skills is chars over four", "skills", transcript.Info{SkillsChars: 38000}, "<t>⚙ 9.5k</>"},
		{"skills under four chars", "skills", transcript.Info{SkillsChars: 3}, "<t>⚙ 0</>"},
		{"skills hidden at zero", "skills", transcript.Info{}, ""},
		{"stripped skills", "skills_stripped", transcript.Info{SkillsBare: 2, SkillsTotal: 9}, "<b>✗ skills 2/9</>"},
		{"stripped skills hidden at zero", "skills_stripped", transcript.Info{SkillsTotal: 9}, ""},
		{"agent skills", "agent_skills", transcript.Info{SubBare: 1, SubTotal: 4}, "<w>✗ agent skills 1/4</>"},
		{"agent skills hidden at zero", "agent_skills", transcript.Info{SubTotal: 4}, ""},
		{"mem", "mem", transcript.Info{MemDropped: true}, "<b>✗ mem</>"},
		{"mem hidden", "mem", transcript.Info{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, ok := segment.Lookup(tt.seg)
			require.True(t, ok)
			assert.Equal(t, tt.want, s.Render(&segment.Context{Config: config.New(""), Theme: th, Transcript: tt.info}))
		})
	}
}

func TestModelOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		id     string
		actual string
		want   string
	}{
		{"same family", "claude-opus-4-7", "claude-opus-4-7-20260101", "Opus 4.7 [high]"},
		{"different family", "claude-opus-4-7", "claude-haiku-4-5", "Haiku (skill)"},
		{"every family has a label", "claude-opus-4-7", "claude-sonnet-5", "Sonnet (skill)"},
		{"fable", "claude-opus-4-7", "fable-5", "Fable (skill)"},
		{"mythos", "claude-opus-4-7", "mythos-1", "Mythos (skill)"},
		{"unknown actual family is ignored", "claude-opus-4-7", "something-else", "Opus 4.7 [high]"},
		{"no actual model", "claude-opus-4-7", "", "Opus 4.7 [high]"},
		{"unknown session family differs from a known one", "custom-model", "claude-haiku-4-5", "Haiku (skill)"},
		{"unknown on both sides", "custom-model", "other", "Opus 4.7 [high]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := payload.Parse([]byte(`{"model":{"id":"` + tt.id + `","display_name":"Opus 4.7"},"effort":{"level":"high"}}`))
			m, ok := segment.Lookup("model")
			require.True(t, ok)
			got := m.Render(&segment.Context{Payload: p, Config: config.New(""), Theme: theme.Lookup("default").WithoutColor(), Transcript: transcript.Info{ActualModel: tt.actual}})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDiagnosticsAreUsageGroup(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"boot", "skills", "skills_stripped", "agent_skills", "mem"} {
		s, ok := segment.Lookup(name)
		require.True(t, ok, name)
		assert.Equal(t, segment.Registry["tokens"].Group(), s.Group(), name)
	}
}

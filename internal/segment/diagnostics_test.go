package segment_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/segment"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/internal/transcript"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Equal(t, tt.want, s.Render(&segment.Context{Config: config.New("").Section(""), Theme: th, Transcript: tt.info}))
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

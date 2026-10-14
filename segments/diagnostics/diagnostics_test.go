package diagnostics_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/internal/transcript"
	"github.com/PaddleHQ/claude-statusline/segments/diagnostics"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func build(t *testing.T, name string, info transcript.Info) plugin.Segment {
	t.Helper()
	providers := plugin.NewProviders(&plugin.Sources{
		Config:     config.New(""),
		Theme:      theme.Theme{Text: "<t>", Bad: "<b>", Warn: "<w>", Reset: "</>"},
		Transcript: func() transcript.Info { return info },
	})
	for _, e := range diagnostics.Plugin().Segments {
		if e.Name == name {
			return e.New(providers.For(name))
		}
	}
	require.Failf(t, "not provided", "%s", name)
	return nil
}

func TestRender(t *testing.T) {
	t.Parallel()

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
			assert.Equal(t, tt.want, build(t, tt.seg, tt.info).Render(&plugin.Session{}))
		})
	}
}

func TestProvidesTheUsageGroupSegments(t *testing.T) {
	t.Parallel()

	entries := diagnostics.Plugin().Segments
	require.Len(t, entries, 5)
	for _, e := range entries {
		seg := build(t, e.Name, transcript.Info{})
		assert.Equal(t, e.Name, seg.Name())
		assert.Equal(t, layout.Usage, seg.Group(), e.Name)
	}
}

func TestEverySegmentHasHelp(t *testing.T) {
	t.Parallel()

	for _, e := range diagnostics.Plugin().Segments {
		assert.NotEmpty(t, build(t, e.Name, transcript.Info{}).Help(), e.Name)
	}
}

package diagnostics_test

import (
	"strconv"
	"testing"

	"github.com/PaddleHQ/claude-statusline/segments/diagnostics"
	"github.com/PaddleHQ/claude-statusline/segments/diagnostics/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// info is the transcript a test describes; every field is zero by default.
type info struct {
	boot                                 int64
	skillsChars, skillsBare, skillsTotal int
	subBare, subTotal                    int
	memDropped                           bool
}

func build(t *testing.T, name string, i info) (seg *diagnostics.Diagnostic, reads *int) {
	t.Helper()

	tr := mocks.NewTranscript(t)
	tr.EXPECT().BootTokens().Return(i.boot).Maybe()
	tr.EXPECT().SkillListChars().Return(i.skillsChars).Maybe()
	tr.EXPECT().SkillCounts().Return(i.skillsBare, i.skillsTotal).Maybe()
	tr.EXPECT().SubagentSkillCounts().Return(i.subBare, i.subTotal).Maybe()
	tr.EXPECT().MemoryDropped().Return(i.memDropped).Maybe()

	paint := mocks.NewPainter(t)
	paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return "<t>" + s + "</>" }).Maybe()
	paint.EXPECT().Bad(mock.Anything).RunAndReturn(func(s string) string { return "<b>" + s + "</>" }).Maybe()
	paint.EXPECT().Warn(mock.Anything).RunAndReturn(func(s string) string { return "<w>" + s + "</>" }).Maybe()

	count := 0
	seg, ok := diagnostics.New(name, diagnostics.Deps{
		Transcript: func() diagnostics.Transcript { count++; return tr },
		Paint:      paint,
		Tokens:     func(n int64) string { return "T" + strconv.FormatInt(n, 10) },
	})
	require.True(t, ok, name)
	return seg, &count
}

func TestRender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		seg  string
		info info
		want string
	}{
		{"boot", "boot", info{boot: 64506}, "<t>⌂ T64506</>"},
		{"boot hidden at zero", "boot", info{}, ""},
		{"skills is chars over four", "skills", info{skillsChars: 38000}, "<t>⚙ T9500</>"},
		{"skills of a three entry listing", "skills", info{skillsChars: 64}, "<t>⚙ T16</>"},
		{"skills under four chars", "skills", info{skillsChars: 3}, "<t>⚙ T0</>"},
		{"skills hidden at zero", "skills", info{}, ""},
		{"stripped skills", "skills_stripped", info{skillsBare: 2, skillsTotal: 9}, "<b>✗ skills 2/9</>"},
		{"stripped skills hidden at zero", "skills_stripped", info{skillsTotal: 9}, ""},
		{"agent skills", "agent_skills", info{subBare: 1, subTotal: 4}, "<w>✗ agent skills 1/4</>"},
		{"agent skills hidden at zero", "agent_skills", info{subTotal: 4}, ""},
		{"mem", "mem", info{memDropped: true}, "<b>✗ mem</>"},
		{"mem hidden", "mem", info{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			seg, _ := build(t, tt.seg, tt.info)
			assert.Equal(t, tt.want, seg.Render())
		})
	}
}

func TestProvidesFiveSegments(t *testing.T) {
	t.Parallel()

	names := diagnostics.Names()
	assert.Equal(t, []string{"boot", "skills", "skills_stripped", "agent_skills", "mem"}, names)
	for _, name := range names {
		seg, reads := build(t, name, info{})
		assert.Equal(t, name, seg.Name())
		assert.NotEmpty(t, seg.Help(), name)
		assert.Empty(t, seg.Keys(), name)
		assert.Zero(t, *reads, "%s asked for the transcript before drawing", name)
	}
}

func TestUnknownSegment(t *testing.T) {
	t.Parallel()

	_, ok := diagnostics.New("nope", diagnostics.Deps{})
	assert.False(t, ok)
}

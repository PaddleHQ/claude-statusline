package model_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/segments/model"
	"github.com/PaddleHQ/claude-statusline/segments/model/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type session struct {
	display, id, level, path string
}

// build returns the segment and a counter of how often the transcript was asked for.
func build(t *testing.T, s session, effort bool, actual string) (seg *model.Model, reads *int) {
	t.Helper()

	sess := mocks.NewSession(t)
	sess.EXPECT().DisplayName().Return(s.display).Maybe()
	sess.EXPECT().ModelID().Return(s.id).Maybe()
	sess.EXPECT().EffortLevel().Return(s.level).Maybe()
	sess.EXPECT().TranscriptFile().Return(s.path).Maybe()

	tr := mocks.NewTranscript(t)
	tr.EXPECT().LatestModel().Return(actual).Maybe()

	cfg := mocks.NewConfig(t)
	cfg.EXPECT().Bool("effort").Return(effort).Maybe()

	paint := mocks.NewPainter(t)
	paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return "<t>" + s + "</t>" }).Maybe()

	count := 0
	return model.New(model.Deps{
		Session:    sess,
		Transcript: func() model.Transcript { count++; return tr },
		Config:     cfg,
		Paint:      paint,
	}), &count
}

func TestRender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		s      session
		effort bool
		want   string
	}{
		{"name and effort", session{display: "Opus 4.7", level: "high"}, true, "<t>Opus 4.7 [high]</t>"},
		{"no effort", session{display: "Sonnet 5.5"}, true, "<t>Sonnet 5.5</t>"},
		{"effort hidden", session{display: "Opus 4.7", level: "high"}, false, "<t>Opus 4.7</t>"},
		{"no model", session{level: "high"}, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			seg, _ := build(t, tt.s, tt.effort, "")
			assert.Equal(t, tt.want, seg.Render())
		})
	}
}

func TestSkillOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		id     string
		actual string
		want   string
	}{
		{"same family", "claude-opus-4-7", "claude-opus-4-7-20260101", "<t>Opus 4.7 [high]</t>"},
		{"different family", "claude-opus-4-7", "claude-haiku-4-5", "<t>Haiku (skill)</t>"},
		{"every family has a label", "claude-opus-4-7", "claude-sonnet-5", "<t>Sonnet (skill)</t>"},
		{"fable", "claude-opus-4-7", "fable-5", "<t>Fable (skill)</t>"},
		{"mythos", "claude-opus-4-7", "mythos-1", "<t>Mythos (skill)</t>"},
		{"unknown actual family is ignored", "claude-opus-4-7", "something-else", "<t>Opus 4.7 [high]</t>"},
		{"no actual model", "claude-opus-4-7", "", "<t>Opus 4.7 [high]</t>"},
		{"unknown session family differs from a known one", "custom-model", "claude-haiku-4-5", "<t>Haiku (skill)</t>"},
		{"unknown on both sides", "custom-model", "other", "<t>Opus 4.7 [high]</t>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			seg, _ := build(t, session{display: "Opus 4.7", id: tt.id, level: "high", path: "/t.jsonl"}, true, tt.actual)
			assert.Equal(t, tt.want, seg.Render())
		})
	}
}

func TestReadsTheTranscriptOnlyWhenThePayloadNamesOne(t *testing.T) {
	t.Parallel()

	seg, reads := build(t, session{display: "Opus 4.7", id: "claude-opus-4-7"}, true, "claude-haiku-4-5")
	assert.Equal(t, "<t>Opus 4.7</t>", seg.Render())
	assert.Zero(t, *reads, "no transcript path, so no read")

	seg, reads = build(t, session{display: "Opus 4.7", id: "claude-opus-4-7", path: "/t.jsonl"}, true, "")
	seg.Render()
	assert.Equal(t, 1, *reads)
}

func TestNameAndKeys(t *testing.T) {
	t.Parallel()

	seg, reads := build(t, session{}, true, "")
	assert.Equal(t, "model", seg.Name())
	assert.Equal(t, []string{"effort"}, seg.Keys())
	assert.Zero(t, *reads)
}

func TestHelp(t *testing.T) {
	t.Parallel()

	withEffort, _ := build(t, session{}, true, "")
	withoutEffort, _ := build(t, session{}, false, "")
	assert.Contains(t, withEffort.Help(), "effort")
	assert.NotContains(t, withoutEffort.Help(), "effort")
	assert.Contains(t, withoutEffort.Help(), "model")
}

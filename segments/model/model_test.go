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
		name    string
		id      string
		display string
		actual  string
		want    string
	}{
		{"same family", "claude-opus-4-7", "Opus 4.7", "claude-opus-4-7-20260101", "<t>Opus 4.7 [high]</t>"},
		{"different family", "claude-opus-4-7", "Opus 4.7", "claude-haiku-4-5", "<t>Haiku 4.5 (skill)</t>"},
		{"another family", "claude-opus-4-7", "Opus 4.7", "claude-sonnet-5", "<t>Sonnet 5 (skill)</t>"},
		{"fable", "claude-opus-4-7", "Opus 4.7", "claude-fable-5", "<t>Fable 5 (skill)</t>"},
		{"mythos", "claude-opus-4-7", "Opus 4.7", "claude-mythos-1", "<t>Mythos 1 (skill)</t>"},
		{"a family nobody listed is labeled from its id", "claude-opus-4-7", "Opus 4.7", "claude-sonic-hedgehog-3-1", "<t>Sonic Hedgehog 3.1 (skill)</t>"},
		{"a family with no version is labeled from its id", "claude-opus-4-7", "Opus 4.7", "claude-sonic-hedgehog", "<t>Sonic Hedgehog (skill)</t>"},
		{"a context suffix on the session id is the same family", "claude-opus-5-5[1m]", "Opus 5.5", "claude-opus-5-5", "<t>Opus 5.5 [high]</t>"},
		{"a Bedrock session id is the same family as the plain id", "us.anthropic.claude-sonnet-4-5-20250929-v1:0", "Sonnet 4.5", "claude-sonnet-4-5-20250929", "<t>Sonnet 4.5 [high]</t>"},
		{"a Bedrock session id still differs from another family", "us.anthropic.claude-sonnet-4-5-20250929-v1:0", "Sonnet 4.5", "claude-haiku-4-5", "<t>Haiku 4.5 (skill)</t>"},
		{"a latest alias session id is the same family as the dated transcript id", "claude-3-7-sonnet-latest", "Sonnet 3.7", "claude-3-7-sonnet-20250219", "<t>Sonnet 3.7 [high]</t>"},
		{"an inference profile ARN session id is the same family as the plain id", "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-sonnet-4-5-20250929-v1:0", "Sonnet 4.5", "claude-sonnet-4-5-20250929", "<t>Sonnet 4.5 [high]</t>"},
		{"a legacy transcript id is labeled", "claude-opus-4-7", "Opus 4.7", "claude-3-5-haiku-20241022", "<t>Haiku 3.5 (skill)</t>"},
		{"a new version of the same family is not a skill", "claude-opus-4-7", "Opus 4.7", "claude-opus-5-5", "<t>Opus 4.7 [high]</t>"},
		{"unknown actual family is ignored", "claude-opus-4-7", "Opus 4.7", "something-else", "<t>Opus 4.7 [high]</t>"},
		{"no actual model", "claude-opus-4-7", "Opus 4.7", "", "<t>Opus 4.7 [high]</t>"},
		{"unknown session family differs from a known one", "custom-model", "Custom Model", "claude-haiku-4-5", "<t>Haiku 4.5 (skill)</t>"},
		{"unknown on both sides", "custom-model", "Custom Model", "other", "<t>Custom Model [high]</t>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			seg, _ := build(t, session{display: tt.display, id: tt.id, level: "high", path: "/t.jsonl"}, true, tt.actual)
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

func TestLabel(t *testing.T) {
	t.Parallel()

	tests := []struct{ name, id, want string }{
		{"family and version", "claude-opus-5-5", "Opus 5.5"},
		{"a date suffix is dropped", "claude-haiku-4-5-20251001", "Haiku 4.5"},
		{"a major version only has no dot", "claude-opus-4-20250514", "Opus 4"},
		{"a family nobody listed is capitalised word by word", "claude-sonic-hedgehog-3-1", "Sonic Hedgehog 3.1"},
		{"a family with no version is shown without one", "claude-sonic-hedgehog", "Sonic Hedgehog"},
		{"a legacy id puts the version first", "claude-3-5-sonnet-20241022", "Sonnet 3.5"},
		{"a context suffix is dropped", "claude-opus-5-5[1m]", "Opus 5.5"},
		{"a Vertex date is dropped", "claude-sonnet-4-5@20250929", "Sonnet 4.5"},
		{"an unknown suffix keeps the id unparsed", "claude-opus-4-7-preview", "opus-4-7-preview"},
		{"a latest alias is the same model in the legacy order", "claude-3-7-sonnet-latest", "Sonnet 3.7"},
		{"a latest alias is the same model in the modern order", "claude-opus-4-latest", "Opus 4"},
		{"a latest alias with a context suffix", "claude-3-5-sonnet-latest[1m]", "Sonnet 3.5"},
		{"a Bedrock inference profile ARN", "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-sonnet-4-5-20250929-v1:0", "Sonnet 4.5"},
		{"a Bedrock foundation model ARN", "arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-3-5-sonnet-20241022-v2:0", "Sonnet 3.5"},
		{"an opaque application inference profile ARN stays unparsed", "arn:aws:bedrock:us-east-1:123:application-inference-profile/abc123", "arn:aws:bedrock:us-east-1:123:application-inference-profile/abc123"},
		{"a bare alias is shown as it is", "sonnet", "sonnet"},
		{"a Bedrock id loses its provider prefix and version", "us.anthropic.claude-sonnet-4-5-20250929-v1:0", "Sonnet 4.5"},
		{"an empty id stays empty", "", ""},
		{"only the prefix leaves nothing", "claude-", ""},
		{"upper case is not a modern id", "Claude-Opus-4-7", "Claude-Opus-4-7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, model.Label(tt.id))
		})
	}
}

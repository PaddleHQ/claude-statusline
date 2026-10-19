package model_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/segments/model"
	"github.com/PaddleHQ/claude-statusline/segments/model/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func build(t *testing.T, display, level string, effort bool) *model.Model {
	t.Helper()

	sess := mocks.NewSession(t)
	sess.EXPECT().DisplayName().Return(display).Maybe()
	sess.EXPECT().EffortLevel().Return(level).Maybe()

	cfg := mocks.NewConfig(t)
	cfg.EXPECT().Bool("effort").Return(effort).Maybe()

	paint := mocks.NewPainter(t)
	paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return "<t>" + s + "</t>" }).Maybe()

	return model.New(model.Deps{Session: sess, Config: cfg, Paint: paint})
}

func TestRender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		display string
		level   string
		effort  bool
		want    string
	}{
		{"name and effort", "Opus 4.7", "high", true, "<t>Opus 4.7 [high]</t>"},
		{"no effort", "Sonnet 5.5", "", true, "<t>Sonnet 5.5</t>"},
		{"effort hidden", "Opus 4.7", "high", false, "<t>Opus 4.7</t>"},
		{"no model", "", "high", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, build(t, tt.display, tt.level, tt.effort).Render())
		})
	}
}

func TestNameAndKeys(t *testing.T) {
	t.Parallel()

	seg := build(t, "", "", true)
	assert.Equal(t, "model", seg.Name())
	assert.Equal(t, []string{"effort"}, seg.Keys())
}

package model_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/segments/model"
	"github.com/PaddleHQ/claude-statusline/segments/model/mocks"

	"github.com/stretchr/testify/assert"
)

func TestRender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		display string
		level   string
		want    string
	}{
		{"name and effort", "Opus 4.7", "high", "Opus 4.7 [high]"},
		{"no effort", "Sonnet 5.5", "", "Sonnet 5.5"},
		{"no model", "", "high", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sess := mocks.NewSession(t)
			sess.EXPECT().DisplayName().Return(tt.display)
			sess.EXPECT().EffortLevel().Return(tt.level).Maybe()

			assert.Equal(t, tt.want, model.New(model.Deps{Session: sess}).Render())
		})
	}
}

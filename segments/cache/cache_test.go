package cache_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/segments/cache"
	"github.com/PaddleHQ/claude-statusline/segments/cache/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func build(t *testing.T, input, creation, read int64, warn int) *cache.Cache {
	t.Helper()

	sess := mocks.NewSession(t)
	sess.EXPECT().LatestRequest().Return(input, creation, read).Maybe()
	cfg := mocks.NewConfig(t)
	cfg.EXPECT().Int("warn").Return(warn).Maybe()
	paint := mocks.NewPainter(t)
	paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return "<t>" + s + "</>" }).Maybe()
	paint.EXPECT().Bad(mock.Anything).RunAndReturn(func(s string) string { return "<b>" + s + "</>" }).Maybe()
	return cache.New(cache.Deps{Session: sess, Config: cfg, Paint: paint})
}

func TestRender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		input, create, read int64
		warn                int
		want                string
	}{
		{"healthy", 10, 10, 80, 5, "<t>⧉ 80%</>"},
		{"at the threshold is not red", 95, 0, 5, 5, "<t>⧉ 5%</>"},
		{"below the threshold is red", 96, 0, 4, 5, "<b>⧉ 4%</>"},
		{"three percent is red", 50, 50, 4, 5, "<b>⧉ 3%</>"},
		{"integer division truncates", 1, 0, 2, 0, "<t>⧉ 66%</>"},
		{"custom threshold", 10, 10, 80, 90, "<b>⧉ 80%</>"},
		{"all reads is 100", 0, 0, 10, 5, "<t>⧉ 100%</>"},
		{"zero denominator hides the segment", 0, 0, 0, 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, build(t, tt.input, tt.create, tt.read, tt.warn).Render())
		})
	}
}

func TestNameAndKeys(t *testing.T) {
	t.Parallel()

	seg := build(t, 0, 0, 0, 5)
	assert.Equal(t, "cache", seg.Name())
	assert.Equal(t, []string{"warn"}, seg.Keys())
}

func TestHelp(t *testing.T) {
	t.Parallel()

	assert.Contains(t, build(t, 0, 0, 0, 12).Help(), "12%")
}

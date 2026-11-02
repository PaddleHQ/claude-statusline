package ctx_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/segments/ctx"
	"github.com/PaddleHQ/claude-statusline/segments/ctx/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func build(t *testing.T, used float64, reported bool, warn, bad int) (*ctx.Context, *mocks.Meter) {
	t.Helper()

	sess := mocks.NewSession(t)
	sess.EXPECT().UsedPercentage().Return(used, reported).Maybe()
	cfg := mocks.NewConfig(t)
	cfg.EXPECT().Int("warn").Return(warn).Maybe()
	cfg.EXPECT().Int("bad").Return(bad).Maybe()
	meter := mocks.NewMeter(t)
	meter.EXPECT().Label(mock.Anything).RunAndReturn(func(s string) string { return "<t>" + s + ": </>" }).Maybe()
	return ctx.New(ctx.Deps{Session: sess, Config: cfg, Meter: meter}), meter
}

func TestRender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		used      float64
		wantUsage int
	}{
		{"whole number", 49, 49},
		{"rounds to even down", 42.5, 42},
		{"rounds to even up", 43.5, 44},
		{"zero is shown", 0, 0},
		{"a fraction rounds to the nearest", 64.6, 65},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			seg, meter := build(t, tt.used, true, 50, 90)
			meter.EXPECT().Usage(tt.wantUsage, 50, 90).Return("BAR").Once()
			assert.Equal(t, "<t>ctx: </>BAR", seg.Render())
		})
	}
}

func TestRenderPassesTheConfiguredThresholds(t *testing.T) {
	t.Parallel()

	seg, meter := build(t, 20, true, 10, 20)
	meter.EXPECT().Usage(20, 10, 20).Return("BAR").Once()
	assert.Equal(t, "<t>ctx: </>BAR", seg.Render())
}

func TestAbsentUsageHidesTheSegment(t *testing.T) {
	t.Parallel()

	seg, _ := build(t, 0, false, 50, 90)
	assert.Empty(t, seg.Render())
}

func TestNameKeysAndHelp(t *testing.T) {
	t.Parallel()

	seg, _ := build(t, 0, false, 40, 80)
	assert.Equal(t, "ctx", seg.Name())
	assert.Equal(t, []string{"width", "width_medium", "width_narrow", "warn", "bad"}, seg.Keys())
	assert.Contains(t, seg.Help(), "40%")
	assert.Contains(t, seg.Help(), "80%")
}

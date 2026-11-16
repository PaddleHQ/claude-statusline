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
	paint := mocks.NewPainter(t)
	paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return "<t>" + s + "</>" }).Maybe()
	meter := mocks.NewMeter(t)
	icons := mocks.NewIcons(t)
	icons.EXPECT().Pick(mock.Anything, mock.Anything).RunAndReturn(func(plain, _ string) string { return plain }).Maybe()
	return ctx.New(ctx.Deps{Session: sess, Config: cfg, Paint: paint, Meter: meter, Icons: icons}), meter
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

func TestContextNerdIcon(t *testing.T) {
	t.Parallel()

	sess := mocks.NewSession(t)
	sess.EXPECT().UsedPercentage().Return(49, true)
	cfg := mocks.NewConfig(t)
	cfg.EXPECT().Int(mock.Anything).Return(50)
	paint := mocks.NewPainter(t)
	paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return s })
	meter := mocks.NewMeter(t)
	meter.EXPECT().Usage(mock.Anything, mock.Anything, mock.Anything).Return("BAR")
	icons := mocks.NewIcons(t)
	icons.EXPECT().Pick("ctx: ", "\U000F035B ").RunAndReturn(func(_, nerd string) string { return nerd })
	seg := ctx.New(ctx.Deps{Session: sess, Config: cfg, Paint: paint, Meter: meter, Icons: icons})
	assert.Equal(t, "\U000F035B BAR", seg.Render())
}

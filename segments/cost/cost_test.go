package cost_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/segments/cost"
	"github.com/PaddleHQ/claude-statusline/segments/cost/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func build(t *testing.T, usd float64, reported bool, warn, bad int) *cost.Cost {
	t.Helper()

	sess := mocks.NewSession(t)
	sess.EXPECT().TotalCost().Return(usd, reported).Maybe()
	cfg := mocks.NewConfig(t)
	cfg.EXPECT().Int("warn").Return(warn).Maybe()
	cfg.EXPECT().Int("bad").Return(bad).Maybe()
	paint := mocks.NewPainter(t)
	paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return "<t>" + s + "</>" }).Maybe()
	paint.EXPECT().Warn(mock.Anything).RunAndReturn(func(s string) string { return "<w>" + s + "</>" }).Maybe()
	paint.EXPECT().Bad(mock.Anything).RunAndReturn(func(s string) string { return "<b>" + s + "</>" }).Maybe()
	return cost.New(cost.Deps{Session: sess, Config: cfg, Paint: paint})
}

func TestRender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		usd       float64
		reported  bool
		warn, bad int
		want      string
	}{
		{"not reported hides the segment", 0, false, 5, 20, ""},
		{"zero is shown", 0, true, 5, 20, "<t>$0.00</>"},
		{"rounds to cents", 1.234, true, 5, 20, "<t>$1.23</>"},
		{"rounds up", 0.129, true, 5, 20, "<t>$0.13</>"},
		{"just below warn", 4.99, true, 5, 20, "<t>$4.99</>"},
		{"rounding up to warn is amber", 4.999, true, 5, 20, "<w>$5.00</>"},
		{"rounding down to below warn is not amber", 4.994, true, 5, 20, "<t>$4.99</>"},
		{"rounding up to bad is red", 19.996, true, 5, 20, "<b>$20.00</>"},
		{"at warn is amber", 5, true, 5, 20, "<w>$5.00</>"},
		{"just below bad", 19.99, true, 5, 20, "<w>$19.99</>"},
		{"at bad is red", 20, true, 5, 20, "<b>$20.00</>"},
		{"large", 1234.5, true, 5, 20, "<b>$1234.50</>"},
		{"custom thresholds", 2, true, 1, 2, "<b>$2.00</>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, build(t, tt.usd, tt.reported, tt.warn, tt.bad).Render())
		})
	}
}

func TestNameAndKeys(t *testing.T) {
	t.Parallel()

	seg := build(t, 0, false, 5, 20)
	assert.Equal(t, "cost", seg.Name())
	assert.Equal(t, []string{"warn", "bad"}, seg.Keys())
}

func TestHelp(t *testing.T) {
	t.Parallel()

	help := build(t, 0, false, 7, 30).Help()
	assert.Contains(t, help, "$7")
	assert.Contains(t, help, "$30")
}

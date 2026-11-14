package ctx_test

import (
	"math"
	"testing"

	"github.com/PaddleHQ/claude-statusline/segments/ctx"
	"github.com/PaddleHQ/claude-statusline/segments/ctx/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type session struct {
	used     float64
	reported bool
	window   int64
	hasSize  bool
	tokens   int64
}

type settings struct {
	warn, bad, compactAt, compactPct int
}

var defaults = settings{warn: 50, bad: 90}

func build(t *testing.T, s session, c settings) (*ctx.Context, *mocks.Meter) {
	t.Helper()

	sess := mocks.NewSession(t)
	sess.EXPECT().UsedPercentage().Return(s.used, s.reported).Maybe()
	sess.EXPECT().WindowSize().Return(s.window, s.hasSize).Maybe()
	sess.EXPECT().LatestRequest().Return(s.tokens/2, s.tokens/4, s.tokens-s.tokens/2-s.tokens/4).Maybe()
	cfg := mocks.NewConfig(t)
	cfg.EXPECT().Int("warn").Return(c.warn).Maybe()
	cfg.EXPECT().Int("bad").Return(c.bad).Maybe()
	cfg.EXPECT().Int("compact_at").Return(c.compactAt).Maybe()
	cfg.EXPECT().Int("compact_pct").Return(c.compactPct).Maybe()
	paint := mocks.NewPainter(t)
	paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return "<t>" + s + "</>" }).Maybe()
	paint.EXPECT().Bad(mock.Anything).RunAndReturn(func(s string) string { return "<b>" + s + "</>" }).Maybe()
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
			seg, meter := build(t, session{used: tt.used, reported: true}, defaults)
			meter.EXPECT().Usage(tt.wantUsage, 50, 90).Return("BAR").Once()
			assert.Equal(t, "<t>ctx: </>BAR", seg.Render())
		})
	}
}

func TestRenderPassesTheConfiguredThresholds(t *testing.T) {
	t.Parallel()

	seg, meter := build(t, session{used: 20, reported: true}, settings{warn: 10, bad: 20})
	meter.EXPECT().Usage(20, 10, 20).Return("BAR").Once()
	assert.Equal(t, "<t>ctx: </>BAR", seg.Render())
}

func TestRenderCompactMarker(t *testing.T) {
	t.Parallel()

	both := settings{warn: 50, bad: 90, compactAt: 200000, compactPct: 80}
	tests := []struct {
		name      string
		s         session
		c         settings
		wantLimit int
		wantReach bool
		wantUsage bool
	}{
		{
			name:      "a 1M window puts the marker at the 200k limit, below it",
			s:         session{used: 15, reported: true, window: 1000000, hasSize: true, tokens: 150000},
			c:         both,
			wantLimit: 20,
		},
		{
			name:      "a 1M window warns at the 200k limit",
			s:         session{used: 21, reported: true, window: 1000000, hasSize: true, tokens: 210000},
			c:         both,
			wantLimit: 20,
			wantReach: true,
		},
		{
			name:      "a 200k window puts the marker at 80 percent, below it",
			s:         session{used: 75, reported: true, window: 200000, hasSize: true, tokens: 150000},
			c:         both,
			wantLimit: 80,
		},
		{
			name:      "a 200k window warns from 160k",
			s:         session{used: 82.5, reported: true, window: 200000, hasSize: true, tokens: 165000},
			c:         both,
			wantLimit: 80,
			wantReach: true,
		},
		{
			name:      "reaching the limit exactly warns",
			s:         session{used: 80, reported: true, window: 200000, hasSize: true, tokens: 160000},
			c:         both,
			wantLimit: 80,
			wantReach: true,
		},
		{
			name:      "a missing window size is inferred from the used percentage",
			s:         session{used: 21, reported: true, tokens: 210000},
			c:         both,
			wantLimit: 20,
			wantReach: true,
		},
		{
			name:      "a zero window size is inferred too",
			s:         session{used: 21, reported: true, window: 0, hasSize: true, tokens: 210000},
			c:         both,
			wantLimit: 20,
			wantReach: true,
		},
		{
			name:      "no window at all keeps only the absolute limit, reached",
			s:         session{used: 0, reported: true, tokens: 210000},
			c:         both,
			wantLimit: -1,
			wantReach: true,
		},
		{
			name:      "no window at all keeps only the absolute limit, not reached",
			s:         session{used: 0, reported: true, tokens: 1000},
			c:         both,
			wantLimit: -1,
		},
		{
			name:      "a limit at or past the window draws no marker but still warns",
			s:         session{used: 90, reported: true, window: 150000, hasSize: true, tokens: 210000},
			c:         settings{warn: 50, bad: 90, compactAt: 200000},
			wantLimit: 133,
			wantReach: true,
		},
		{
			name:      "only the percentage limit applies when compact_at is off",
			s:         session{used: 50, reported: true, window: 1000000, hasSize: true, tokens: 500000},
			c:         settings{warn: 50, bad: 90, compactPct: 80},
			wantLimit: 80,
		},
		{
			name:      "only the absolute limit applies when compact_pct is off",
			s:         session{used: 50, reported: true, window: 400000, hasSize: true, tokens: 200000},
			c:         settings{warn: 50, bad: 90, compactAt: 200000},
			wantLimit: 50,
			wantReach: true,
		},
		{
			name:      "the percentage limit alone has nothing to go on without a window",
			s:         session{used: 0, reported: true},
			c:         settings{warn: 50, bad: 90, compactPct: 80},
			wantUsage: true,
		},
		{
			name:      "both limits off behaves as before",
			s:         session{used: 95, reported: true, window: 1000000, hasSize: true, tokens: 950000},
			c:         defaults,
			wantUsage: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			seg, meter := build(t, tt.s, tt.c)
			pct := int(math.RoundToEven(tt.s.used))
			want := "<t>ctx: </>BAR"
			if tt.wantUsage {
				meter.EXPECT().Usage(pct, 50, 90).Return("BAR").Once()
			} else {
				meter.EXPECT().Limit(pct, tt.wantLimit, tt.wantReach, 50, 90).Return("BAR").Once()
			}
			if tt.wantReach {
				want += " <b>⚠ /compact</>"
			}
			assert.Equal(t, want, seg.Render())
		})
	}
}

func TestAbsentUsageHidesTheSegment(t *testing.T) {
	t.Parallel()

	seg, _ := build(t, session{}, defaults)
	assert.Empty(t, seg.Render())
}

func TestNameKeysAndHelp(t *testing.T) {
	t.Parallel()

	seg, _ := build(t, session{}, settings{warn: 40, bad: 80, compactAt: 150000, compactPct: 70})
	assert.Equal(t, "ctx", seg.Name())
	assert.Equal(t, []string{"width", "width_medium", "width_narrow", "warn", "bad", "compact_at", "compact_pct"}, seg.Keys())
	assert.Contains(t, seg.Help(), "40%")
	assert.Contains(t, seg.Help(), "80%")
	assert.Contains(t, seg.Help(), "150000 tokens")
	assert.Contains(t, seg.Help(), "70%")
}

func TestHelpDescribesOnlyTheLimitsThatAreOn(t *testing.T) {
	t.Parallel()

	base := "How full the context window is. Amber from 50%, red from 90%."
	tests := []struct {
		name string
		at   int
		pct  int
		want string
	}{
		{"tokens only", 200000, 0, base + " A marker shows where to /compact, at 200000 tokens, and a red warning once you reach it."},
		{"percentage only", 0, 80, base + " A marker shows where to /compact, at 80% of the window, and a red warning once you reach it."},
		{"both off", 0, 0, base},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			seg, _ := build(t, session{}, settings{warn: 50, bad: 90, compactAt: tt.at, compactPct: tt.pct})
			assert.Equal(t, tt.want, seg.Help())
		})
	}
}

func TestContextNerdIcons(t *testing.T) {
	t.Parallel()

	sess := mocks.NewSession(t)
	sess.EXPECT().UsedPercentage().Return(21, true)
	sess.EXPECT().WindowSize().Return(1000000, true)
	sess.EXPECT().LatestRequest().Return(0, 0, 210000)
	cfg := mocks.NewConfig(t)
	cfg.EXPECT().Int("compact_at").Return(200000)
	cfg.EXPECT().Int("compact_pct").Return(80)
	cfg.EXPECT().Int("warn").Return(50)
	cfg.EXPECT().Int("bad").Return(90)
	paint := mocks.NewPainter(t)
	paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return s })
	paint.EXPECT().Bad(mock.Anything).RunAndReturn(func(s string) string { return s })
	meter := mocks.NewMeter(t)
	meter.EXPECT().Limit(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("BAR")
	icons := mocks.NewIcons(t)
	icons.EXPECT().Pick("ctx: ", "\U000F035B ").RunAndReturn(func(_, nerd string) string { return nerd })
	icons.EXPECT().Pick("⚠ /compact", " /compact").RunAndReturn(func(_, nerd string) string { return nerd })
	seg := ctx.New(ctx.Deps{Session: sess, Config: cfg, Paint: paint, Meter: meter, Icons: icons})
	assert.Equal(t, "\U000F035B BAR  /compact", seg.Render())
}

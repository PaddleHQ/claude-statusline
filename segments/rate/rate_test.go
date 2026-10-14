package rate_test

import (
	"testing"
	"time"

	"github.com/PaddleHQ/claude-statusline/segments/rate"
	"github.com/PaddleHQ/claude-statusline/segments/rate/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const now = 1760000000

type fixture struct {
	limits *mocks.Limits
	meter  *mocks.Meter
	deps   rate.Deps
	reads  int
}

func newFixture(t *testing.T, cfg map[string]string) *fixture {
	t.Helper()

	f := &fixture{limits: mocks.NewLimits(t), meter: mocks.NewMeter(t)}
	config := mocks.NewConfig(t)
	config.EXPECT().String("label").Return(cfg["label"]).Maybe()
	for key, value := range map[string]int{"warn": 20, "bad": 40, "hard_cap": 95} {
		config.EXPECT().Int(key).Return(value).Maybe()
	}
	paint := mocks.NewPainter(t)
	paint.EXPECT().Text(mock.Anything).RunAndReturn(func(s string) string { return "<t>" + s + "</>" }).Maybe()
	f.deps = rate.Deps{
		Config: config,
		Paint:  paint,
		Meter:  f.meter,
		Now:    time.Unix(now, 0),
		Limits: func() rate.Limits { f.reads++; return f.limits },
	}
	return f
}

func TestFiveHourDrawsItsReading(t *testing.T) {
	t.Parallel()

	f := newFixture(t, map[string]string{"label": "5h"})
	f.limits.EXPECT().FiveHourReading().Return(45, now+9000, true)
	f.meter.EXPECT().Pace(45, 50, 20, 40, 95).Return("BAR 45%")

	assert.Equal(t, "<t>5h: </>BAR 45%", rate.NewFiveHour(f.deps).Render())
}

func TestSevenDayDrawsItsReading(t *testing.T) {
	t.Parallel()

	f := newFixture(t, map[string]string{"label": "7d"})
	f.limits.EXPECT().SevenDayReading().Return(40, now+302400, true)
	f.meter.EXPECT().Pace(40, 50, 20, 40, 95).Return("BAR 40%")

	assert.Equal(t, "<t>7d: </>BAR 40%", rate.NewSevenDay(f.deps).Render())
}

func TestTargetIsTheElapsedShareOfTheWindowClamped(t *testing.T) {
	t.Parallel()

	const window = 5 * 60 * 60
	tests := []struct {
		name     string
		resetsAt int64
		want     int
	}{
		{"just started", now + window, 0},
		{"before the window", now + window + 1000, 0},
		{"after the window", now - 1000, 100},
		{"rounds down", now + window - 179, 0},
		{"one percent", now + window - 180, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t, map[string]string{"label": "5h"})
			f.limits.EXPECT().FiveHourReading().Return(10, tt.resetsAt, true)
			f.meter.EXPECT().Pace(10, tt.want, 20, 40, 95).Return("BAR").Once()
			rate.NewFiveHour(f.deps).Render()
		})
	}
}

func TestNoReadingHidesTheSegment(t *testing.T) {
	t.Parallel()

	f := newFixture(t, map[string]string{"label": "5h"})
	f.limits.EXPECT().FiveHourReading().Return(0, 0, false)
	assert.Empty(t, rate.NewFiveHour(f.deps).Render())
}

func TestLimitsAreOnlyAskedForWhenDrawing(t *testing.T) {
	t.Parallel()

	f := newFixture(t, nil)
	seg := rate.NewFiveHour(f.deps)
	seg.Name()
	seg.Keys()
	seg.Help()
	assert.Zero(t, f.reads)
}

func TestNamesAndKeys(t *testing.T) {
	t.Parallel()

	f := newFixture(t, nil)
	for name, seg := range map[string]*rate.Window{"rate5h": rate.NewFiveHour(f.deps), "rate7d": rate.NewSevenDay(f.deps)} {
		assert.Equal(t, name, seg.Name())
		assert.Len(t, seg.Keys(), 7, "%s declares %v", name, seg.Keys())
	}
}

func TestHelp(t *testing.T) {
	t.Parallel()

	f := newFixture(t, nil)
	five := rate.NewFiveHour(f.deps).Help()
	seven := rate.NewSevenDay(f.deps).Help()
	for _, want := range []string{"5 hour", "20 points", "40 points", "95%"} {
		assert.Contains(t, five, want)
	}
	assert.Contains(t, seven, "7 day")
}

func TestLabelIsDrawnAsText(t *testing.T) {
	t.Parallel()

	f := newFixture(t, map[string]string{"label": "`touch x`; $(y)"})
	f.limits.EXPECT().SevenDayReading().Return(40, now+302400, true)
	f.meter.EXPECT().Pace(40, 50, 20, 40, 95).Return("BAR 40%")

	assert.Equal(t, "<t>`touch x`; $(y): </>BAR 40%", rate.NewSevenDay(f.deps).Render())
}

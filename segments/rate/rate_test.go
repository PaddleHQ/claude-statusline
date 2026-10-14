package rate_test

import (
	"testing"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments/rate"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func build(t *testing.T, name, cfg string, rates ratecache.Limits, now time.Time, columns int) plugin.Segment {
	t.Helper()
	classic := theme.Load("classic", "", "")
	providers := plugin.NewProviders(&plugin.Sources{
		Config:     config.New(cfg, rate.Plugin().Defaults),
		Theme:      classic.WithoutColor(),
		Now:        now,
		Columns:    columns,
		RateLimits: func() ratecache.Limits { return rates },
	})
	for _, e := range rate.Plugin().Segments {
		if e.Name == name {
			return e.New(providers.For(name))
		}
	}
	require.Failf(t, "not provided", "%s", name)
	return nil
}

func TestRender(t *testing.T) {
	t.Parallel()

	now := time.Unix(1760000000, 0)
	rates := ratecache.Limits{
		FiveHour: &ratecache.Reading{Used: 45, ResetsAt: 1760009000},
		SevenDay: &ratecache.Reading{Used: 40, ResetsAt: 1760302400},
	}

	tests := []struct {
		name    string
		seg     string
		cfg     string
		rates   ratecache.Limits
		columns int
		want    string
	}{
		{"five hour bar", "rate5h", "rate5h.width = 4", rates, 999, "5h: █▊┃░ 45%"},
		{"seven day bar", "rate7d", "rate7d.width = 4", rates, 999, "7d: █▋┃░ 40%"},
		{"custom label", "rate5h", "rate5h.width = 4\nrate5h.label = five", rates, 999, "five: █▊┃░ 45%"},
		{"narrow terminal shrinks the bar", "rate5h", "rate5h.width_narrow = 4", rates, 100, "5h: █▊┃░ 45%"},
		{"no reading hides the segment", "rate5h", "", ratecache.Limits{}, 999, ""},
		{"only the other window", "rate5h", "", ratecache.Limits{SevenDay: rates.SevenDay}, 999, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			seg := build(t, tt.seg, tt.cfg, tt.rates, now, tt.columns)
			assert.Equal(t, tt.want, seg.Render(&plugin.Session{}))
		})
	}
}

func TestProvidesBothWindowsInTheRatesGroup(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"rate5h", "rate7d"} {
		seg := build(t, name, "", ratecache.Limits{}, time.Time{}, 999)
		assert.Equal(t, name, seg.Name())
		assert.Equal(t, layout.Rates, seg.Group())
		assert.Len(t, seg.Keys(), 7, "%s declares %v", name, seg.Keys())
	}
}

func TestHelp(t *testing.T) {
	t.Parallel()

	cfg := "rate5h.warn = 11\nrate5h.bad = 22\nrate5h.hard_cap = 88\nrate7d.warn = 33\nrate7d.bad = 44\nrate7d.hard_cap = 99"
	five := build(t, "rate5h", cfg, ratecache.Limits{}, time.Time{}, 999).Help()
	seven := build(t, "rate7d", cfg, ratecache.Limits{}, time.Time{}, 999).Help()
	for _, want := range []string{"5 hour", "11 points", "22 points", "88%"} {
		assert.Contains(t, five, want)
	}
	for _, want := range []string{"7 day", "33 points", "44 points", "99%"} {
		assert.Contains(t, seven, want)
	}
}

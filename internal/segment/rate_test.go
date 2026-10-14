package segment_test

import (
	"strings"
	"testing"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/segment"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateRender(t *testing.T) {
	t.Parallel()

	now := time.Unix(1760000000, 0)
	rates := ratecache.Limits{
		FiveHour: &ratecache.Reading{Used: 45, ResetsAt: 1760009000},
		SevenDay: &ratecache.Reading{Used: 40, ResetsAt: 1760302400},
	}
	plain := plainClassic()

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
			s, ok := segment.Lookup(tt.seg)
			require.True(t, ok)
			ctx := &segment.Context{Rates: tt.rates, Config: config.New(tt.cfg).Section(""), Theme: plain, Now: now, Columns: tt.columns}
			assert.Equal(t, tt.want, s.Render(ctx))
		})
	}
}

func TestRateKeysArePrefixed(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"rate5h", "rate7d"} {
		s, ok := segment.Lookup(name)
		require.True(t, ok, "%s is not registered", name)
		assert.Len(t, s.Keys(), 7, "%s declares %v", name, s.Keys())
		for _, k := range s.Keys() {
			assert.True(t, strings.HasPrefix(k, name+"."), "%s declares %q", name, k)
		}
	}
}

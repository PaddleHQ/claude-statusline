package ratecache_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
)

func val(used, resets int64) ratecache.Reading {
	return ratecache.Reading{Used: used, ResetsAt: resets}
}

func r(used, resets int64) *ratecache.Reading {
	v := val(used, resets)
	return &v
}

func TestIsNewer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		in     ratecache.Reading
		cached *ratecache.Reading
		want   bool
	}{
		{"empty cache", val(45, 100), nil, true},
		{"later window", val(5, 200), r(90, 100), true},
		{"earlier window", val(90, 100), r(5, 200), false},
		{"same window, higher usage", val(55, 100), r(45, 100), true},
		{"same window, same usage", val(45, 100), r(45, 100), false},
		{"same window, small drop", val(35, 100), r(45, 100), false},
		{"same window, drop of exactly ten", val(35, 100), r(45, 100), false},
		{"same window, drop of eleven", val(34, 100), r(45, 100), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ratecache.IsNewer(tt.in, tt.cached))
		})
	}
}

func TestMerge(t *testing.T) {
	t.Parallel()

	cached := ratecache.Limits{FiveHour: r(45, 100), SevenDay: r(40, 500)}

	t.Run("windows merge independently", func(t *testing.T) {
		t.Parallel()
		got, changed := ratecache.Merge(cached, ratecache.Limits{FiveHour: r(40, 100), SevenDay: r(41, 500)})
		assert.True(t, changed)
		assert.Equal(t, ratecache.Limits{FiveHour: r(45, 100), SevenDay: r(41, 500)}, got)
	})
	t.Run("nothing incoming keeps the cache", func(t *testing.T) {
		t.Parallel()
		got, changed := ratecache.Merge(cached, ratecache.Limits{})
		assert.False(t, changed)
		assert.Equal(t, cached, got)
	})
	t.Run("nothing newer reports no change", func(t *testing.T) {
		t.Parallel()
		_, changed := ratecache.Merge(cached, ratecache.Limits{FiveHour: r(40, 100)})
		assert.False(t, changed, "Merge() reported a change")
	})
	t.Run("fills an empty window only", func(t *testing.T) {
		t.Parallel()
		got, changed := ratecache.Merge(ratecache.Limits{FiveHour: r(1, 2)}, ratecache.Limits{SevenDay: r(3, 4)})
		assert.True(t, changed)
		assert.Equal(t, ratecache.Limits{FiveHour: r(1, 2), SevenDay: r(3, 4)}, got)
	})
}

func TestVisible(t *testing.T) {
	t.Parallel()

	got := ratecache.Limits{FiveHour: r(1, 100), SevenDay: r(2, 101)}.Visible(100)
	assert.Nil(t, got.FiveHour, "a window resetting exactly now should be hidden")
	assert.NotNil(t, got.SevenDay, "a window resetting later should stay")
}

func f(v float64) *float64 { return &v }

func TestFromPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   payload.Window
		want *ratecache.Reading
	}{
		{"rounds usage to nearest", payload.Window{UsedPercentage: f(45.5), ResetsAt: f(100)}, r(46, 100)},
		{"rounds down below half", payload.Window{UsedPercentage: f(45.4), ResetsAt: f(100)}, r(45, 100)},
		{"floors the reset time", payload.Window{UsedPercentage: f(1), ResetsAt: f(100.9)}, r(1, 100)},
		{"missing usage", payload.Window{ResetsAt: f(100)}, nil},
		{"missing reset", payload.Window{UsedPercentage: f(1)}, nil},
		{"too large", payload.Window{UsedPercentage: f(1), ResetsAt: f(1e13)}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ratecache.FromPayload(payload.RateLimits{FiveHour: tt.in}).FiveHour
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadTolerance(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write := func(content string) string {
		path := filepath.Join(dir, "c")
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
		return path
	}

	tests := []struct {
		name    string
		content string
		want    ratecache.Limits
	}{
		{"valid", `{"five_hour":{"used":1,"resets_at":2},"seven_day":{"used":3,"resets_at":4}}`, ratecache.Limits{FiveHour: r(1, 2), SevenDay: r(3, 4)}},
		{"corrupt", `{"five_hour":`, ratecache.Limits{}},
		{"not an object", `[1]`, ratecache.Limits{}},
		{"empty", ``, ratecache.Limits{}},
		{"half entry", `{"five_hour":{"used":1},"seven_day":{"used":3,"resets_at":4}}`, ratecache.Limits{SevenDay: r(3, 4)}},
		{"negative values", `{"five_hour":{"used":-1,"resets_at":2}}`, ratecache.Limits{}},
		{"wrong types", `{"five_hour":{"used":"1","resets_at":2}}`, ratecache.Limits{}},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ratecache.Load(write(tt.content)), tt.name)
	}
	assert.Equal(t, ratecache.Limits{}, ratecache.Load(filepath.Join(dir, "missing")))
}

func TestStoreAndSync(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "cache", "ratelimits")

	got := ratecache.Sync(path, ratecache.Limits{FiveHour: r(45, 100)})
	require.Equal(t, ratecache.Limits{FiveHour: r(45, 100)}, got)
	assert.Equal(t, got, ratecache.Load(path))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	before := info.ModTime()
	require.NoError(t, os.Chtimes(path, before.Add(-1e9*3600), before.Add(-1e9*3600)))
	old, err := os.Stat(path)
	require.NoError(t, err)
	ratecache.Sync(path, ratecache.Limits{FiveHour: r(40, 100)})
	after, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, after.ModTime().Equal(old.ModTime()), "an unchanged merge rewrote the cache")

	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	assert.Len(t, entries, 1, "temporary files left behind: %v", entries)
}

func TestSyncWithoutPath(t *testing.T) {
	t.Parallel()

	got := ratecache.Sync("", ratecache.Limits{SevenDay: r(1, 2)})
	assert.Equal(t, ratecache.Limits{SevenDay: r(1, 2)}, got)
}

func TestStoreUnwritableIsSilent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blocker := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))
	ratecache.Store(filepath.Join(blocker, "sub", "ratelimits"), ratecache.Limits{FiveHour: r(1, 2)})
}

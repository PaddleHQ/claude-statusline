package ratecache_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		got, changed := cached.Merge(ratecache.Limits{FiveHour: r(40, 100), SevenDay: r(41, 500)})
		assert.True(t, changed)
		assert.Equal(t, ratecache.Limits{FiveHour: r(45, 100), SevenDay: r(41, 500)}, got)
	})
	t.Run("nothing incoming keeps the cache", func(t *testing.T) {
		t.Parallel()
		got, changed := cached.Merge(ratecache.Limits{})
		assert.False(t, changed)
		assert.Equal(t, cached, got)
	})
	t.Run("nothing newer reports no change", func(t *testing.T) {
		t.Parallel()
		_, changed := cached.Merge(ratecache.Limits{FiveHour: r(40, 100)})
		assert.False(t, changed, "Merge() reported a change")
	})
	t.Run("fills an empty window only", func(t *testing.T) {
		t.Parallel()
		empty := ratecache.Limits{FiveHour: r(1, 2)}
		got, changed := empty.Merge(ratecache.Limits{SevenDay: r(3, 4)})
		assert.True(t, changed)
		assert.Equal(t, ratecache.Limits{FiveHour: r(1, 2), SevenDay: r(3, 4)}, got)
	})
}

func TestVisible(t *testing.T) {
	t.Parallel()

	limits := ratecache.Limits{FiveHour: r(1, 100), SevenDay: r(2, 101)}
	got := limits.Visible(100)
	assert.Nil(t, got.FiveHour, "a window resetting exactly now should be hidden")
	assert.NotNil(t, got.SevenDay, "a window resetting later should stay")
}

func TestFromPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   payload.Window
		want *ratecache.Reading
	}{
		{"rounds usage to nearest", payload.Window{UsedPercentage: new(45.5), ResetsAt: new(100.0)}, r(46, 100)},
		{"rounds down below half", payload.Window{UsedPercentage: new(45.4), ResetsAt: new(100.0)}, r(45, 100)},
		{"floors the reset time", payload.Window{UsedPercentage: new(1.0), ResetsAt: new(100.9)}, r(1, 100)},
		{"missing usage", payload.Window{ResetsAt: new(100.0)}, nil},
		{"missing reset", payload.Window{UsedPercentage: new(1.0)}, nil},
		{"too large", payload.Window{UsedPercentage: new(1.0), ResetsAt: new(1e13)}, nil},
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
		assert.Equal(t, tt.want, ratecache.NewCache(write(tt.content)).Load(), tt.name)
	}
	assert.Equal(t, ratecache.Limits{}, ratecache.NewCache(filepath.Join(dir, "missing")).Load())
}

func TestStoreAndSync(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "cache", "ratelimits")

	got := ratecache.NewCache(path).Sync(ratecache.Limits{FiveHour: r(45, 100)})
	require.Equal(t, ratecache.Limits{FiveHour: r(45, 100)}, got)
	assert.Equal(t, got, ratecache.NewCache(path).Load())

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	before := info.ModTime()
	require.NoError(t, os.Chtimes(path, before.Add(-1e9*3600), before.Add(-1e9*3600)))
	old, err := os.Stat(path)
	require.NoError(t, err)
	ratecache.NewCache(path).Sync(ratecache.Limits{FiveHour: r(40, 100)})
	after, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, after.ModTime().Equal(old.ModTime()), "an unchanged merge rewrote the cache")

	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	assert.Len(t, entries, 1, "temporary files left behind: %v", entries)
}

func TestSyncWithoutPath(t *testing.T) {
	t.Parallel()

	got := ratecache.NewCache("").Sync(ratecache.Limits{SevenDay: r(1, 2)})
	assert.Equal(t, ratecache.Limits{SevenDay: r(1, 2)}, got)
}

func TestStoreUnwritableIsSilent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blocker := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))
	ratecache.NewCache(filepath.Join(blocker, "sub", "ratelimits")).Store(ratecache.Limits{FiveHour: r(1, 2)})
}

func TestSyncWritesTheMergedReadings(t *testing.T) {
	t.Parallel()

	const cached = `{"five_hour":{"used":45,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}`
	tests := []struct {
		name     string
		cached   string
		incoming ratecache.Limits
		want     string
	}{
		{"the first reading is written", "", ratecache.Limits{FiveHour: r(45, 1760009000), SevenDay: r(40, 1760302400)}, cached},
		{"nothing in the payload keeps the cache as it is", cached, ratecache.Limits{}, cached},
		{
			"a higher reading replaces the cached one", cached,
			ratecache.Limits{FiveHour: r(55, 1760009000), SevenDay: r(40, 1760302400)},
			`{"five_hour":{"used":55,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}`,
		},
		{"a lower reading from an idle session keeps the cached one", cached, ratecache.Limits{FiveHour: r(40, 1760009000), SevenDay: r(38, 1760302400)}, cached},
		{"a drop of exactly ten points is stale", cached, ratecache.Limits{FiveHour: r(35, 1760009000), SevenDay: r(40, 1760302400)}, cached},
		{
			"a drop of more than ten points is a real reset", cached,
			ratecache.Limits{FiveHour: r(20, 1760009000), SevenDay: r(40, 1760302400)},
			`{"five_hour":{"used":20,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}`,
		},
		{"a reading from an earlier window is ignored", cached, ratecache.Limits{FiveHour: r(5, 1759991000), SevenDay: r(40, 1760302400)}, cached},
		{
			"a reading from a later window replaces a higher one", `{"five_hour":{"used":90,"resets_at":1759991000},"seven_day":{"used":40,"resets_at":1760302400}}`,
			ratecache.Limits{FiveHour: r(5, 1760009000), SevenDay: r(40, 1760302400)},
			`{"five_hour":{"used":5,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "ratelimits")
			if tt.cached != "" {
				require.NoError(t, os.WriteFile(path, []byte(tt.cached+"\n"), 0o600))
			}
			ratecache.NewCache(path).Sync(tt.incoming)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(data))
		})
	}
}

func TestVisibleHidesAWindowThatHasReset(t *testing.T) {
	t.Parallel()

	limits := ratecache.Limits{FiveHour: r(45, 1759999900), SevenDay: r(40, 1760302400)}
	got := limits.Visible(1760000000)
	assert.Nil(t, got.FiveHour, "the five hour window reset 100 seconds ago")
	assert.Equal(t, r(40, 1760302400), got.SevenDay)
}

func TestLimitsReadings(t *testing.T) {
	t.Parallel()

	l := ratecache.Limits{FiveHour: r(45, 9000)}
	used, resets, ok := l.FiveHourReading()
	assert.True(t, ok)
	assert.Equal(t, []int64{45, 9000}, []int64{used, resets})
	_, _, ok = l.SevenDayReading()
	assert.False(t, ok)

	l = ratecache.Limits{SevenDay: r(10, 20)}
	used, resets, ok = l.SevenDayReading()
	assert.True(t, ok)
	assert.Equal(t, []int64{10, 20}, []int64{used, resets})
}

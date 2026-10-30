package ratecache_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

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
		{"sessions are not part of the limits", `{"five_hour":{"used":1,"resets_at":2},"sessions":{"a":{"five_hour":{"used":9,"resets_at":2}}}}`, ratecache.Limits{FiveHour: r(1, 2)}},
		{"corrupt sessions keep the readings", `{"five_hour":{"used":1,"resets_at":2},"sessions":"x"}`, ratecache.Limits{FiveHour: r(1, 2)}},
		{"corrupt session entries keep the readings", `{"seven_day":{"used":3,"resets_at":4},"sessions":{"a":[1],"b":null}}`, ratecache.Limits{SevenDay: r(3, 4)}},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ratecache.NewCache(write(tt.content)).Load(), tt.name)
	}
	assert.Equal(t, ratecache.Limits{}, ratecache.NewCache(filepath.Join(dir, "missing")).Load())
}

func TestMerge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cached   *ratecache.Reading
		incoming *ratecache.Reading
		previous *ratecache.Reading
		want     *ratecache.Reading
	}{
		{"nothing incoming keeps the cache", r(45, 100), nil, nil, r(45, 100)},
		{"empty cache takes the incoming", nil, r(45, 100), nil, r(45, 100)},
		{"later window wins", r(90, 100), r(5, 200), r(90, 100), r(5, 200)},
		{"earlier window loses", r(5, 200), r(90, 100), r(1, 100), r(5, 200)},
		{"same window, higher usage", r(45, 100), r(55, 100), nil, r(55, 100)},
		{"same window, same usage", r(45, 100), r(45, 100), r(30, 100), r(45, 100)},
		{"drop from a first sighting", r(45, 100), r(24, 100), nil, r(45, 100)},
		{"drop repeating the previous reading", r(45, 100), r(24, 100), r(24, 100), r(45, 100)},
		{"drop of a single point repeating the previous reading", r(45, 100), r(44, 100), r(44, 100), r(45, 100)},
		{"drop after the session's reading changed", r(45, 100), r(24, 100), r(30, 100), r(24, 100)},
		{"drop after the session's reading changed by a single point", r(45, 100), r(44, 100), r(45, 100), r(44, 100)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cached := ratecache.Limits{FiveHour: tt.cached}
			got, changed := cached.Merge(ratecache.Limits{FiveHour: tt.incoming}, ratecache.Limits{FiveHour: tt.previous})
			assert.Equal(t, tt.want, got.FiveHour)
			assert.Equal(t, tt.cached == nil || *tt.cached != *tt.want, changed)
		})
	}

	t.Run("windows merge independently", func(t *testing.T) {
		t.Parallel()
		cached := ratecache.Limits{FiveHour: r(45, 100), SevenDay: r(40, 500)}
		got, changed := cached.Merge(ratecache.Limits{FiveHour: r(40, 100), SevenDay: r(41, 500)}, ratecache.Limits{})
		assert.True(t, changed)
		assert.Equal(t, ratecache.Limits{FiveHour: r(45, 100), SevenDay: r(41, 500)}, got)
	})
	t.Run("fills an empty window only", func(t *testing.T) {
		t.Parallel()
		empty := ratecache.Limits{FiveHour: r(1, 2)}
		got, changed := empty.Merge(ratecache.Limits{SevenDay: r(3, 4)}, ratecache.Limits{})
		assert.True(t, changed)
		assert.Equal(t, ratecache.Limits{FiveHour: r(1, 2), SevenDay: r(3, 4)}, got)
	})
}

func TestStoreAndSync(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "cache", "ratelimits")

	got := ratecache.NewCache(path).Sync("a", ratecache.Limits{FiveHour: r(45, 100)})
	require.Equal(t, ratecache.Limits{FiveHour: r(45, 100)}, got)
	assert.Equal(t, got, ratecache.NewCache(path).Load())

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	lock, err := os.Stat(path + ".lock")
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), lock.Mode().Perm())

	before := info.ModTime()
	require.NoError(t, os.Chtimes(path, before.Add(-1e9*3600), before.Add(-1e9*3600)))
	old, err := os.Stat(path)
	require.NoError(t, err)
	ratecache.NewCache(path).Sync("a", ratecache.Limits{FiveHour: r(45, 100)})
	ratecache.NewCache(path).Sync("b", ratecache.Limits{FiveHour: r(40, 100)})
	ratecache.NewCache(path).Sync("b", ratecache.Limits{FiveHour: r(40, 100)})
	after, err := os.Stat(path)
	require.NoError(t, err)
	assert.False(t, after.ModTime().Equal(old.ModTime()), "a new session was not recorded")

	require.NoError(t, os.Chtimes(path, before.Add(-1e9*3600), before.Add(-1e9*3600)))
	old, err = os.Stat(path)
	require.NoError(t, err)
	ratecache.NewCache(path).Sync("a", ratecache.Limits{FiveHour: r(45, 100)})
	ratecache.NewCache(path).Sync("b", ratecache.Limits{FiveHour: r(40, 100)})
	ratecache.NewCache(path).Sync("", ratecache.Limits{FiveHour: r(30, 100)})
	after, err = os.Stat(path)
	require.NoError(t, err)
	assert.True(t, after.ModTime().Equal(old.ModTime()), "an unchanged merge rewrote the cache")

	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.ElementsMatch(t, []string{"ratelimits", "ratelimits.lock"}, names, "temporary files left behind")
}

func TestSyncWithoutPath(t *testing.T) {
	t.Parallel()

	got := ratecache.NewCache("").Sync("a", ratecache.Limits{SevenDay: r(1, 2)})
	assert.Equal(t, ratecache.Limits{SevenDay: r(1, 2)}, got)
}

func TestSyncWithoutLock(t *testing.T) {
	t.Parallel()

	t.Run("a lock file that cannot be opened", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "ratelimits")
		require.NoError(t, os.Mkdir(path+".lock", 0o700))
		got := ratecache.NewCache(path).Sync("a", ratecache.Limits{FiveHour: r(45, 100)})
		assert.Equal(t, ratecache.Limits{FiveHour: r(45, 100)}, got)
		assert.Equal(t, got, ratecache.NewCache(path).Load())
	})
	t.Run("an unwritable directory", func(t *testing.T) {
		t.Parallel()
		blocker := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(blocker, nil, 0o600))
		got := ratecache.NewCache(filepath.Join(blocker, "sub", "ratelimits")).Sync("a", ratecache.Limits{FiveHour: r(45, 100)})
		assert.Equal(t, ratecache.Limits{FiveHour: r(45, 100)}, got)
	})
	t.Run("a lock held by a stalled session", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "ratelimits")
		held, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
		require.NoError(t, err)
		t.Cleanup(func() { _ = held.Close() })
		require.NoError(t, syscall.Flock(int(held.Fd()), syscall.LOCK_EX)) //nolint:gosec // G115: a file descriptor fits in an int

		start := time.Now()
		got := ratecache.NewCache(path).Sync("a", ratecache.Limits{FiveHour: r(45, 100)})
		assert.Less(t, time.Since(start), 3*time.Second, "Sync waited on a held lock")
		assert.Equal(t, ratecache.Limits{FiveHour: r(45, 100)}, got)
		assert.Equal(t, got, ratecache.NewCache(path).Load())
	})
}

func TestSyncFreshnessBySession(t *testing.T) {
	t.Parallel()

	type step struct {
		session  string
		incoming ratecache.Limits
	}
	tests := []struct {
		name  string
		steps []step
		want  string
	}{
		{
			"an idle session repeating a stale reading never lowers the cache",
			[]step{
				{"a", ratecache.Limits{FiveHour: r(71, 100)}},
				{"b", ratecache.Limits{FiveHour: r(24, 100)}},
				{"b", ratecache.Limits{FiveHour: r(24, 100)}},
				{"b", ratecache.Limits{FiveHour: r(24, 100)}},
			},
			`{"five_hour":{"used":71,"resets_at":100},"sessions":{"a":{"five_hour":{"used":71,"resets_at":100}},"b":{"five_hour":{"used":24,"resets_at":100}}}}`,
		},
		{
			"two idle sessions taking turns never lower the cache",
			[]step{
				{"a", ratecache.Limits{FiveHour: r(71, 100), SevenDay: r(83, 500)}},
				{"b", ratecache.Limits{FiveHour: r(9, 100), SevenDay: r(59, 500)}},
				{"c", ratecache.Limits{FiveHour: r(24, 100), SevenDay: r(59, 500)}},
				{"b", ratecache.Limits{FiveHour: r(9, 100), SevenDay: r(59, 500)}},
				{"c", ratecache.Limits{FiveHour: r(24, 100), SevenDay: r(59, 500)}},
			},
			`{"five_hour":{"used":71,"resets_at":100},"seven_day":{"used":83,"resets_at":500},"sessions":{` +
				`"a":{"five_hour":{"used":71,"resets_at":100},"seven_day":{"used":83,"resets_at":500}},` +
				`"b":{"five_hour":{"used":9,"resets_at":100},"seven_day":{"used":59,"resets_at":500}},` +
				`"c":{"five_hour":{"used":24,"resets_at":100},"seven_day":{"used":59,"resets_at":500}}}}`,
		},
		{
			"a session whose reading changed can lower the cache",
			[]step{
				{"a", ratecache.Limits{FiveHour: r(71, 100)}},
				{"a", ratecache.Limits{FiveHour: r(5, 100)}},
			},
			`{"five_hour":{"used":5,"resets_at":100},"sessions":{"a":{"five_hour":{"used":5,"resets_at":100}}}}`,
		},
		{
			"a session sees its own change in one window only",
			[]step{
				{"a", ratecache.Limits{FiveHour: r(71, 100), SevenDay: r(83, 500)}},
				{"a", ratecache.Limits{FiveHour: r(71, 100), SevenDay: r(10, 500)}},
			},
			`{"five_hour":{"used":71,"resets_at":100},"seven_day":{"used":10,"resets_at":500},"sessions":{` +
				`"a":{"five_hour":{"used":71,"resets_at":100},"seven_day":{"used":10,"resets_at":500}}}}`,
		},
		{
			"a session that is first seen cannot lower the cache",
			[]step{
				{"a", ratecache.Limits{FiveHour: r(71, 100)}},
				{"b", ratecache.Limits{FiveHour: r(5, 100)}},
			},
			`{"five_hour":{"used":71,"resets_at":100},"sessions":{"a":{"five_hour":{"used":71,"resets_at":100}},"b":{"five_hour":{"used":5,"resets_at":100}}}}`,
		},
		{
			"a session without an ID cannot lower the cache",
			[]step{
				{"a", ratecache.Limits{FiveHour: r(71, 100)}},
				{"", ratecache.Limits{FiveHour: r(24, 100)}},
				{"", ratecache.Limits{FiveHour: r(5, 100)}},
			},
			`{"five_hour":{"used":71,"resets_at":100},"sessions":{"a":{"five_hour":{"used":71,"resets_at":100}}}}`,
		},
		{
			"a session without an ID can raise the cache",
			[]step{
				{"a", ratecache.Limits{FiveHour: r(71, 100)}},
				{"", ratecache.Limits{FiveHour: r(80, 100)}},
			},
			`{"five_hour":{"used":80,"resets_at":100},"sessions":{"a":{"five_hour":{"used":71,"resets_at":100}}}}`,
		},
		{
			"a later window replaces a higher reading and forgets sessions from the earlier one",
			[]step{
				{"a", ratecache.Limits{FiveHour: r(90, 100)}},
				{"b", ratecache.Limits{FiveHour: r(10, 100)}},
				{"c", ratecache.Limits{FiveHour: r(5, 200)}},
			},
			`{"five_hour":{"used":5,"resets_at":200},"sessions":{"c":{"five_hour":{"used":5,"resets_at":200}}}}`,
		},
		{
			"a reading from an earlier window is ignored and not remembered",
			[]step{
				{"a", ratecache.Limits{FiveHour: r(50, 200)}},
				{"b", ratecache.Limits{FiveHour: r(90, 100)}},
			},
			`{"five_hour":{"used":50,"resets_at":200},"sessions":{"a":{"five_hour":{"used":50,"resets_at":200}}}}`,
		},
		{
			"a session that drops the five hour window is forgotten",
			[]step{
				{"a", ratecache.Limits{FiveHour: r(50, 100), SevenDay: r(40, 500)}},
				{"b", ratecache.Limits{SevenDay: r(45, 500)}},
			},
			`{"five_hour":{"used":50,"resets_at":100},"seven_day":{"used":45,"resets_at":500},"sessions":{` +
				`"a":{"five_hour":{"used":50,"resets_at":100},"seven_day":{"used":40,"resets_at":500}}}}`,
		},
		{
			"sessions follow the seven day window when there is no five hour reading",
			[]step{
				{"a", ratecache.Limits{SevenDay: r(40, 500)}},
				{"b", ratecache.Limits{SevenDay: r(30, 600)}},
			},
			`{"seven_day":{"used":30,"resets_at":600},"sessions":{"b":{"seven_day":{"used":30,"resets_at":600}}}}`,
		},
		{
			"a seven day drop from an idle session is stale without a five hour reading",
			[]step{
				{"a", ratecache.Limits{SevenDay: r(83, 500)}},
				{"b", ratecache.Limits{SevenDay: r(59, 500)}},
				{"b", ratecache.Limits{SevenDay: r(59, 500)}},
			},
			`{"seven_day":{"used":83,"resets_at":500},"sessions":{"a":{"seven_day":{"used":83,"resets_at":500}},"b":{"seven_day":{"used":59,"resets_at":500}}}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "ratelimits")
			for _, s := range tt.steps {
				ratecache.NewCache(path).Sync(s.session, s.incoming)
			}
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(data))
		})
	}
}

func TestSyncReturnsTheMergedReadings(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "ratelimits")
	assert.Equal(t, ratecache.Limits{FiveHour: r(71, 100)}, ratecache.NewCache(path).Sync("a", ratecache.Limits{FiveHour: r(71, 100)}))
	assert.Equal(t, ratecache.Limits{FiveHour: r(71, 100)}, ratecache.NewCache(path).Sync("b", ratecache.Limits{FiveHour: r(24, 100)}))
	assert.Equal(t, ratecache.Limits{FiveHour: r(71, 100)}, ratecache.NewCache(path).Sync("b", ratecache.Limits{FiveHour: r(24, 100)}))
	assert.Equal(t, ratecache.Limits{FiveHour: r(71, 100)}, ratecache.NewCache(path).Sync("c", ratecache.Limits{}))
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
		{"a lower reading from an unseen session keeps the cached one", cached, ratecache.Limits{FiveHour: r(20, 1760009000), SevenDay: r(38, 1760302400)}, cached},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "ratelimits")
			if tt.cached != "" {
				require.NoError(t, os.WriteFile(path, []byte(tt.cached+"\n"), 0o600))
			}
			ratecache.NewCache(path).Sync("", tt.incoming)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(data))
		})
	}
}

func TestSyncDropsInvalidSessionEntries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			"invalid entries",
			`{"five_hour":{"used":71,"resets_at":100},"sessions":{"a":{"five_hour":{"used":71,"resets_at":100}},"b":7,"c":{"five_hour":{"used":-1,"resets_at":100}},"d":null,"e":{"five_hour":"x"}}}`,
			`{"five_hour":{"used":71,"resets_at":100},"sessions":{"a":{"five_hour":{"used":71,"resets_at":100}},"z":{"five_hour":{"used":24,"resets_at":100}}}}`,
		},
		{
			"a sessions key that is not an object",
			`{"five_hour":{"used":71,"resets_at":100},"sessions":"garbage"}`,
			`{"five_hour":{"used":71,"resets_at":100},"sessions":{"z":{"five_hour":{"used":24,"resets_at":100}}}}`,
		},
		{
			"a sessions key that is a list",
			`{"five_hour":{"used":71,"resets_at":100},"sessions":[1,2]}`,
			`{"five_hour":{"used":71,"resets_at":100},"sessions":{"z":{"five_hour":{"used":24,"resets_at":100}}}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "ratelimits")
			require.NoError(t, os.WriteFile(path, []byte(tt.content), 0o600))
			got := ratecache.NewCache(path).Sync("z", ratecache.Limits{FiveHour: r(24, 100)})
			assert.Equal(t, ratecache.Limits{FiveHour: r(71, 100)}, got)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(data))
		})
	}
}

func TestSyncConcurrentSessionsKeepTheHighest(t *testing.T) {
	t.Parallel()

	const sessions = 20
	path := filepath.Join(t.TempDir(), "ratelimits")

	var wg sync.WaitGroup
	for i := range sessions {
		wg.Go(func() {
			ratecache.NewCache(path).Sync(fmt.Sprintf("s%d", i), ratecache.Limits{FiveHour: r(int64(i+1), 100)})
		})
	}
	wg.Wait()

	assert.Equal(t, ratecache.Limits{FiveHour: r(sessions, 100)}, ratecache.NewCache(path).Load())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, sessions, strings.Count(string(data), `"five_hour"`)-1, "every session should be recorded")
}

func TestStoreUnwritableIsSilent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blocker := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))
	ratecache.NewCache(filepath.Join(blocker, "sub", "ratelimits")).Store(ratecache.Limits{FiveHour: r(1, 2)})
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

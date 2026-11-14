package overage_test

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/overage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var mid = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)

func newTracker(t *testing.T) (tr *overage.Tracker, path string) {
	t.Helper()
	path = overage.PathIn(t.TempDir())
	return overage.New(path), path
}

func total(tr *overage.Tracker, at time.Time) float64 {
	_, usd := tr.Month(at)
	return usd
}

func TestPathIn(t *testing.T) {
	t.Parallel()

	assert.Equal(t, filepath.Join("state", "cache", "overage"), overage.PathIn("state"))
}

func TestMonthKeyAndNextReset(t *testing.T) {
	t.Parallel()

	london := time.FixedZone("BST", 3600)
	tests := []struct {
		name      string
		at        time.Time
		key       string
		nextReset time.Time
	}{
		{"mid month", mid, "2026-11", time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)},
		{"december rolls into the next year", time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC), "2026-12", time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"a zone ahead of UTC still uses the UTC month", time.Date(2026, 12, 1, 0, 30, 0, 0, london), "2026-11", time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.key, overage.MonthKey(tt.at))
			assert.True(t, tt.nextReset.Equal(overage.NextReset(tt.at)), overage.NextReset(tt.at))
		})
	}
}

func TestRecordFastPathWritesNothing(t *testing.T) {
	t.Parallel()

	t.Run("a reading under the limit from an unknown session", func(t *testing.T) {
		t.Parallel()
		tr, path := newTracker(t)
		tr.Record("a", 5, false, mid)
		assert.NoFileExists(t, path)
		assert.NoFileExists(t, path+".lock")
	})

	t.Run("a reading under the limit after one that was under it", func(t *testing.T) {
		t.Parallel()
		tr, path := newTracker(t)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		const content = `{"months":{},"sessions":{"a":{"cost":1,"over":false,"seen":1}}}`
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
		tr.Record("a", 5, false, mid)
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.JSONEq(t, content, string(got))
		assert.NoFileExists(t, path+".lock")
	})
}

func TestRecordNoOps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		session string
		cost    float64
	}{
		{"an empty path has no file", "", "a", 1},
		{"an empty session is ignored", "x", "", 1},
		{"a negative cost is ignored", "x", "a", -1},
		{"NaN is ignored", "x", "a", math.NaN()},
		{"infinity is ignored", "x", "a", math.Inf(1)},
		{"an absurd cost is ignored", "x", "a", 1e13},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := tt.path
			if path != "" {
				path = overage.PathIn(dir)
			}
			overage.New(path).Record(tt.session, tt.cost, true, mid)
			assert.NoFileExists(t, overage.PathIn(dir))
		})
	}

	t.Run("an empty path reads as zero", func(t *testing.T) {
		t.Parallel()
		key, usd := overage.New("").Month(mid)
		assert.Equal(t, "2026-11", key)
		assert.Zero(t, usd)
	})
}

func TestRecordCountsOnlyRisesWhileOver(t *testing.T) {
	t.Parallel()

	type step struct {
		session string
		cost    float64
		over    bool
		want    float64
	}
	tests := []struct {
		name  string
		steps []step
	}{
		{"becoming over counts nothing until the cost rises", []step{
			{"a", 10, false, 0},
			{"a", 12, true, 0},
			{"a", 15.5, true, 3.5},
		}},
		{"a rise while over is counted once", []step{
			{"a", 12, true, 0},
			{"a", 15.5, true, 3.5},
			{"a", 15.5, true, 3.5},
		}},
		{"the reading that leaves the limit still counts its rise, and later rises count nothing", []step{
			{"a", 12, true, 0},
			{"a", 14, true, 2},
			{"a", 20, false, 8},
			{"a", 25, false, 8},
		}},
		{"the first reading over the limit after one under it counts nothing", []step{
			{"a", 14, true, 0},
			{"a", 14, false, 0},
			{"a", 30, true, 0},
			{"a", 31, true, 1},
		}},
		{"a lower cost, as after a session reset, adds nothing and becomes the baseline", []step{
			{"a", 20, true, 0},
			{"a", 2, true, 0},
			{"a", 3, true, 1},
		}},
		{"a render arriving out of order keeps the higher baseline", []step{
			{"a", 10, true, 0},
			{"a", 12, true, 2},
			{"a", 11, true, 2},
			{"a", 13, true, 3},
		}},
		{"sessions are tracked separately and summed", []step{
			{"a", 5, true, 0},
			{"b", 100, true, 0},
			{"a", 6, true, 1},
			{"b", 102.5, true, 3.5},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tr, _ := newTracker(t)
			for i, s := range tt.steps {
				tr.Record(s.session, s.cost, s.over, mid)
				assert.InDelta(t, s.want, total(tr, mid), 1e-9, "step %d", i)
			}
		})
	}
}

func TestRecordMonthRollover(t *testing.T) {
	t.Parallel()

	tr, _ := newTracker(t)
	lastMoment := time.Date(2026, 11, 30, 23, 59, 0, 0, time.UTC)
	nextMonth := time.Date(2026, 12, 1, 0, 1, 0, 0, time.UTC)

	tr.Record("a", 10, true, lastMoment)
	tr.Record("a", 12, true, lastMoment)
	tr.Record("a", 17, true, nextMonth)

	assert.InDelta(t, 2.0, total(tr, lastMoment), 1e-9, "the old month is left intact")
	assert.InDelta(t, 5.0, total(tr, nextMonth), 1e-9, "the delta belongs to the month it was seen in")
}

func TestRecordPrunes(t *testing.T) {
	t.Parallel()

	t.Run("sessions unseen for over seven days are dropped", func(t *testing.T) {
		t.Parallel()
		tr, path := newTracker(t)
		old := mid.Add(-7*24*time.Hour - time.Minute)
		recent := mid.Add(-6 * 24 * time.Hour)
		tr.Record("old", 1, true, old)
		tr.Record("recent", 1, true, recent)
		tr.Record("now", 1, true, mid)

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.NotContains(t, string(got), `"old"`)
		assert.Contains(t, string(got), `"recent"`)
		assert.Contains(t, string(got), `"now"`)
	})

	t.Run("only the twelve newest months are kept", func(t *testing.T) {
		t.Parallel()
		tr, path := newTracker(t)
		for i := range 14 {
			at := time.Date(2025, time.Month(i+1), 15, 0, 0, 0, 0, time.UTC)
			tr.Record("a", float64(i*10), true, at)
			tr.Record("a", float64(i*10+1), true, at)
		}
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.NotContains(t, string(got), `"2025-01"`)
		assert.NotContains(t, string(got), `"2025-02"`)
		assert.Contains(t, string(got), `"2025-03"`)
		assert.Contains(t, string(got), `"2026-02"`)
		assert.Zero(t, total(tr, time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)))
	})
}

func TestCorruptFileCountsAsEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    float64
	}{
		{"not JSON", `not json`, 0},
		{"empty file", ``, 0},
		{"wrong types", `{"months":5,"sessions":"x"}`, 0},
		{"a bad month entry is skipped and a good one kept", `{"months":{"2026-11":{"usd":4},"2026-10":"x","nope":{"usd":1}}}`, 4},
		{"a negative total is ignored", `{"months":{"2026-11":{"usd":-4}}}`, 0},
		{"an absurd total is ignored", `{"months":{"2026-11":{"usd":1e30}}}`, 0},
		{"a month without usd is ignored", `{"months":{"2026-11":{}}}`, 0},
		{"a half session entry is ignored", `{"months":{"2026-11":{"usd":4}},"sessions":{"a":{"cost":1}}}`, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tr, path := newTracker(t)
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
			require.NoError(t, os.WriteFile(path, []byte(tt.content), 0o600))
			assert.InDelta(t, tt.want, total(tr, mid), 1e-9)
		})
	}

	t.Run("recording over a corrupt file starts afresh", func(t *testing.T) {
		t.Parallel()
		tr, path := newTracker(t)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(`{"sessions":{"a":{"cost":"x","over":true,"seen":1}}}`), 0o600))
		tr.Record("a", 5, true, mid)
		tr.Record("a", 7, true, mid)
		assert.InDelta(t, 2.0, total(tr, mid), 1e-9)
	})

	t.Run("a missing file reads as zero", func(t *testing.T) {
		t.Parallel()
		tr, _ := newTracker(t)
		assert.Zero(t, total(tr, mid))
	})
}

func TestRecordKeepsTotalWithinBounds(t *testing.T) {
	t.Parallel()

	tr, _ := newTracker(t)
	tr.Record("a", 0, true, mid)
	tr.Record("a", 9e11, true, mid)
	tr.Record("b", 0, true, mid)
	tr.Record("b", 9e11, true, mid)
	assert.InDelta(t, 1e12, total(tr, mid), 1)
}

func TestRecordUnwritableDirectoryIsSilent(t *testing.T) {
	t.Parallel()

	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))
	tr := overage.New(filepath.Join(blocker, "cache", "overage"))
	assert.NotPanics(t, func() {
		tr.Record("a", 1, true, mid)
		tr.Record("a", 2, true, mid)
	})
	assert.Zero(t, total(tr, mid))
}

func TestRecordConcurrentSessionsSumExactly(t *testing.T) {
	t.Parallel()

	const (
		workers = 8
		rounds  = 25
	)
	tr, _ := newTracker(t)
	for w := range workers {
		tr.Record(fmt.Sprintf("s%d", w), 0, true, mid)
	}

	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() {
			id := fmt.Sprintf("s%d", w)
			for r := 1; r <= rounds; r++ {
				tr.Record(id, float64(r), true, mid)
			}
		})
	}
	wg.Wait()

	assert.InDelta(t, float64(workers*rounds), total(tr, mid), 1e-9)
}

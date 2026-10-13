package ratecache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/payload"
)

func r(used, resets int64) *Reading { return &Reading{Used: used, ResetsAt: resets} }

func TestIsNewer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		in     Reading
		cached *Reading
		want   bool
	}{
		{"empty cache", Reading{45, 100}, nil, true},
		{"later window", Reading{5, 200}, r(90, 100), true},
		{"earlier window", Reading{90, 100}, r(5, 200), false},
		{"same window, higher usage", Reading{55, 100}, r(45, 100), true},
		{"same window, same usage", Reading{45, 100}, r(45, 100), false},
		{"same window, small drop", Reading{35, 100}, r(45, 100), false},
		{"same window, drop of exactly ten", Reading{35, 100}, r(45, 100), false},
		{"same window, drop of eleven", Reading{34, 100}, r(45, 100), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isNewer(tt.in, tt.cached); got != tt.want {
				t.Errorf("isNewer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	t.Parallel()

	cached := Limits{FiveHour: r(45, 100), SevenDay: r(40, 500)}

	t.Run("windows merge independently", func(t *testing.T) {
		t.Parallel()
		got, changed := Merge(cached, Limits{FiveHour: r(40, 100), SevenDay: r(41, 500)})
		if !changed || *got.FiveHour != *cached.FiveHour || *got.SevenDay != *r(41, 500) {
			t.Errorf("Merge() = %+v, %v", got, changed)
		}
	})
	t.Run("nothing incoming keeps the cache", func(t *testing.T) {
		t.Parallel()
		got, changed := Merge(cached, Limits{})
		if changed || got != cached {
			t.Errorf("Merge() = %+v, %v", got, changed)
		}
	})
	t.Run("nothing newer reports no change", func(t *testing.T) {
		t.Parallel()
		if _, changed := Merge(cached, Limits{FiveHour: r(40, 100)}); changed {
			t.Error("Merge() reported a change")
		}
	})
	t.Run("fills an empty window only", func(t *testing.T) {
		t.Parallel()
		got, changed := Merge(Limits{FiveHour: r(1, 2)}, Limits{SevenDay: r(3, 4)})
		if !changed || *got.FiveHour != *r(1, 2) || *got.SevenDay != *r(3, 4) {
			t.Errorf("Merge() = %+v, %v", got, changed)
		}
	})
}

func TestVisible(t *testing.T) {
	t.Parallel()

	got := Limits{FiveHour: r(1, 100), SevenDay: r(2, 101)}.Visible(100)
	if got.FiveHour != nil {
		t.Error("a window resetting exactly now should be hidden")
	}
	if got.SevenDay == nil {
		t.Error("a window resetting later should stay")
	}
}

func f(v float64) *float64 { return &v }

func TestFromPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   payload.Window
		want *Reading
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
			got := FromPayload(payload.RateLimits{FiveHour: tt.in}).FiveHour
			switch {
			case tt.want == nil && got != nil, tt.want != nil && (got == nil || *got != *tt.want):
				t.Errorf("FromPayload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadTolerance(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write := func(content string) string {
		path := filepath.Join(dir, "c")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}

	tests := []struct {
		name    string
		content string
		want    Limits
	}{
		{"valid", `{"five_hour":{"used":1,"resets_at":2},"seven_day":{"used":3,"resets_at":4}}`, Limits{r(1, 2), r(3, 4)}},
		{"corrupt", `{"five_hour":`, Limits{}},
		{"not an object", `[1]`, Limits{}},
		{"empty", ``, Limits{}},
		{"half entry", `{"five_hour":{"used":1},"seven_day":{"used":3,"resets_at":4}}`, Limits{nil, r(3, 4)}},
		{"negative values", `{"five_hour":{"used":-1,"resets_at":2}}`, Limits{}},
		{"wrong types", `{"five_hour":{"used":"1","resets_at":2}}`, Limits{}},
	}
	for _, tt := range tests {
		got := Load(write(tt.content))
		if !equal(got, tt.want) {
			t.Errorf("%s: Load() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
	if got := Load(filepath.Join(dir, "missing")); got != (Limits{}) {
		t.Errorf("Load(missing) = %+v", got)
	}
}

func equal(a, b Limits) bool {
	same := func(x, y *Reading) bool { return (x == nil && y == nil) || (x != nil && y != nil && *x == *y) }
	return same(a.FiveHour, b.FiveHour) && same(a.SevenDay, b.SevenDay)
}

func TestStoreAndSync(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "cache", "ratelimits")

	got := Sync(path, Limits{FiveHour: r(45, 100)})
	if !equal(got, Limits{FiveHour: r(45, 100)}) {
		t.Fatalf("Sync() = %+v", got)
	}
	if !equal(Load(path), got) {
		t.Errorf("cache holds %+v", Load(path))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("cache mode = %o, want 600", perm)
	}

	before := info.ModTime()
	if err := os.Chtimes(path, before.Add(-1e9*3600), before.Add(-1e9*3600)); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Stat(path)
	Sync(path, Limits{FiveHour: r(40, 100)})
	after, _ := os.Stat(path)
	if !after.ModTime().Equal(old.ModTime()) {
		t.Error("an unchanged merge rewrote the cache")
	}

	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("temporary files left behind: %v", entries)
	}
}

func TestSyncWithoutPath(t *testing.T) {
	t.Parallel()

	got := Sync("", Limits{SevenDay: r(1, 2)})
	if !equal(got, Limits{SevenDay: r(1, 2)}) {
		t.Errorf("Sync() = %+v", got)
	}
}

func TestStoreUnwritableIsSilent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blocker := filepath.Join(dir, "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	Store(filepath.Join(blocker, "sub", "ratelimits"), Limits{FiveHour: r(1, 2)})
}

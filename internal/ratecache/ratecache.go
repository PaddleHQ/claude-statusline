// Package ratecache shares rate-limit readings between sessions. Limits belong to the
// account, so an idle session's stale payload must not overwrite a fresher reading.
package ratecache

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"

	"github.com/PaddleHQ/claude-statusline/internal/payload"
)

// resetTolerance is the largest same-window drop treated as a stale idle session
// rather than a mid-window quota reset, in percentage points.
const resetTolerance = 10

// maxValue keeps readings within 12 digits, like every other number the status line accepts.
const maxValue = 1e12

// Reading is one window's usage percentage and the Unix time it resets.
type Reading struct {
	Used     int64 `json:"used"`
	ResetsAt int64 `json:"resets_at"`
}

// Limits holds both windows; nil means no reading.
type Limits struct {
	FiveHour *Reading `json:"five_hour,omitempty"`
	SevenDay *Reading `json:"seven_day,omitempty"`
}

// FromPayload converts a payload's windows to whole-number readings. A window
// missing either field, or out of range, has no reading.
func FromPayload(r payload.RateLimits) Limits {
	return Limits{FiveHour: fromWindow(r.FiveHour), SevenDay: fromWindow(r.SevenDay)}
}

func fromWindow(w payload.Window) *Reading {
	if w.UsedPercentage == nil || w.ResetsAt == nil {
		return nil
	}
	used, resets := math.Round(*w.UsedPercentage), math.Floor(*w.ResetsAt)
	if used >= maxValue || resets >= maxValue {
		return nil
	}
	return &Reading{Used: int64(used), ResetsAt: int64(resets)}
}

// Merge keeps each window's reading in l unless the incoming one is newer, and reports
// whether anything changed.
func (l *Limits) Merge(incoming Limits) (Limits, bool) {
	merged := *l
	five, fiveChanged := pick(l.FiveHour, incoming.FiveHour)
	seven, sevenChanged := pick(l.SevenDay, incoming.SevenDay)
	merged.FiveHour, merged.SevenDay = five, seven
	return merged, fiveChanged || sevenChanged
}

func pick(cached, incoming *Reading) (*Reading, bool) {
	if incoming != nil && isNewer(*incoming, cached) {
		return incoming, true
	}
	return cached, false
}

func isNewer(in Reading, cached *Reading) bool {
	switch {
	case cached == nil:
		return true
	case in.ResetsAt != cached.ResetsAt:
		return in.ResetsAt > cached.ResetsAt
	}
	return in.Used > cached.Used || cached.Used-in.Used > resetTolerance
}

// Visible drops windows that have already reset.
func (l *Limits) Visible(now int64) Limits {
	v := *l
	if v.FiveHour != nil && v.FiveHour.ResetsAt <= now {
		v.FiveHour = nil
	}
	if v.SevenDay != nil && v.SevenDay.ResetsAt <= now {
		v.SevenDay = nil
	}
	return v
}

// Cache is the file that shares readings between sessions.
type Cache struct{ path string }

// NewCache returns the Cache stored at path; an empty path means no file.
func NewCache(path string) *Cache { return &Cache{path: path} }

// Load reads the cache; a missing or corrupt file, or a half entry, counts as empty.
func (c *Cache) Load() Limits {
	data, err := os.ReadFile(c.path) //nolint:gosec // G304: the cache path derives from the user's own Claude config directory
	if err != nil {
		return Limits{}
	}
	var raw struct {
		FiveHour *rawReading `json:"five_hour"`
		SevenDay *rawReading `json:"seven_day"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return Limits{}
	}
	return Limits{FiveHour: raw.FiveHour.reading(), SevenDay: raw.SevenDay.reading()}
}

type rawReading struct {
	Used     *int64 `json:"used"`
	ResetsAt *int64 `json:"resets_at"`
}

func (r *rawReading) reading() *Reading {
	if r == nil || r.Used == nil || r.ResetsAt == nil || *r.Used < 0 || *r.ResetsAt < 0 {
		return nil
	}
	return &Reading{Used: *r.Used, ResetsAt: *r.ResetsAt}
}

// Store writes the cache through a temporary file and a rename, so concurrent sessions
// never see a partial file. Failures are ignored: the cache is an optimization.
func (c *Cache) Store(l Limits) {
	data, err := json.Marshal(l)
	if err != nil {
		return
	}
	dir := filepath.Dir(c.path)
	if os.MkdirAll(dir, 0o700) != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, ".ratelimits-*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(append(data, '\n'))
	cerr := tmp.Close()
	if werr != nil || cerr != nil || os.Rename(tmp.Name(), c.path) != nil {
		_ = os.Remove(tmp.Name())
	}
}

// Sync merges the incoming readings into the cache, writing only when something
// changed, and returns the merged limits. An empty path skips the file.
func (c *Cache) Sync(incoming Limits) Limits {
	if c.path == "" {
		empty := Limits{}
		merged, _ := empty.Merge(incoming)
		return merged
	}
	cached := c.Load()
	merged, changed := cached.Merge(incoming)
	if changed {
		c.Store(merged)
	}
	return merged
}

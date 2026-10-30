// Package ratecache shares rate-limit readings between sessions. Limits belong to the
// account, so an idle session re-sending its stale payload must not overwrite a fresher
// reading: a lower reading is accepted only from a session whose own reading changed.
package ratecache

import (
	"encoding/json"
	"maps"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/payload"
)

// maxValue keeps readings within 12 digits, like every other number the status line accepts.
const maxValue = 1e12

// lockWait bounds how long a render waits for another session's lock before syncing
// without it, so a stalled holder can never freeze every status line.
const (
	lockWait = time.Second
	lockPoll = 5 * time.Millisecond
)

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

// FiveHourReading is the five hour window's usage and reset time; false means there is no reading.
func (l *Limits) FiveHourReading() (used, resetsAt int64, ok bool) { return l.FiveHour.values() }

// SevenDayReading is the seven day window's usage and reset time; false means there is no reading.
func (l *Limits) SevenDayReading() (used, resetsAt int64, ok bool) { return l.SevenDay.values() }

func (r *Reading) values() (used, resetsAt int64, ok bool) {
	if r == nil {
		return 0, 0, false
	}
	return r.Used, r.ResetsAt, true
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
// whether anything changed. previous is the incoming session's last reported readings: a
// same-window drop is a quota reset only when the session's own reading differs from it,
// as an idle session repeats the same stale reading on every render.
func (l *Limits) Merge(incoming, previous Limits) (Limits, bool) {
	five, fiveChanged := pick(l.FiveHour, incoming.FiveHour, previous.FiveHour)
	seven, sevenChanged := pick(l.SevenDay, incoming.SevenDay, previous.SevenDay)
	return Limits{FiveHour: five, SevenDay: seven}, fiveChanged || sevenChanged
}

func pick(cached, incoming, previous *Reading) (*Reading, bool) {
	if incoming == nil || !isNewer(*incoming, cached, previous) {
		return cached, false
	}
	return incoming, true
}

func isNewer(in Reading, cached, previous *Reading) bool {
	switch {
	case cached == nil:
		return true
	case in.ResetsAt != cached.ResetsAt:
		return in.ResetsAt > cached.ResetsAt
	case in.Used > cached.Used:
		return true
	case in.Used < cached.Used:
		return previous != nil && *previous != in
	}
	return false
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

type state struct {
	Limits
	Sessions map[string]Limits `json:"sessions,omitempty"`
}

// Load reads the shared readings; a missing or corrupt file, or a half entry, counts as empty.
func (c *Cache) Load() Limits {
	return c.read().Limits
}

func (c *Cache) read() state {
	data, err := os.ReadFile(c.path) //nolint:gosec // G304: the cache path derives from the user's own Claude config directory
	if err != nil {
		return state{}
	}
	var raw struct {
		FiveHour *rawReading     `json:"five_hour"`
		SevenDay *rawReading     `json:"seven_day"`
		Sessions json.RawMessage `json:"sessions"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return state{}
	}
	return state{
		FiveHour: raw.FiveHour.reading(), SevenDay: raw.SevenDay.reading(),
		Sessions: readSessions(raw.Sessions),
	}
}

func readSessions(data json.RawMessage) map[string]Limits {
	var entries map[string]json.RawMessage
	if json.Unmarshal(data, &entries) != nil {
		return nil
	}
	var sessions map[string]Limits
	for id, entry := range entries {
		var raw struct {
			FiveHour *rawReading `json:"five_hour"`
			SevenDay *rawReading `json:"seven_day"`
		}
		if json.Unmarshal(entry, &raw) != nil {
			continue
		}
		l := Limits{FiveHour: raw.FiveHour.reading(), SevenDay: raw.SevenDay.reading()}
		if l.FiveHour == nil && l.SevenDay == nil {
			continue
		}
		if sessions == nil {
			sessions = map[string]Limits{}
		}
		sessions[id] = l
	}
	return sessions
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

// Store writes the readings through a temporary file and a rename, so concurrent sessions
// never see a partial file. Failures are ignored: the cache is an optimization.
func (c *Cache) Store(l Limits) {
	c.write(state{Limits: l})
}

func (c *Cache) write(s state) {
	data, err := json.Marshal(s)
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

// Sync merges the incoming readings of one session into the cache, writing only when
// something changed, and returns the merged limits. An empty path skips the file, and an
// empty session can raise readings but never lower them. Concurrent callers are serialized
// by a lock file beside the cache, when one can be taken.
func (c *Cache) Sync(session string, incoming Limits) Limits {
	if c.path == "" {
		empty := Limits{}
		merged, _ := empty.Merge(incoming, Limits{})
		return merged
	}
	defer c.lock()()

	cached := c.read()
	merged, _ := cached.Merge(incoming, cached.Sessions[session])

	sessions := make(map[string]Limits, len(cached.Sessions)+1)
	maps.Copy(sessions, cached.Sessions)
	if session != "" {
		sessions[session] = incoming
	}
	next := state{Limits: merged, Sessions: prune(sessions, merged)}
	if !reflect.DeepEqual(next, cached) {
		c.write(next)
	}
	return merged
}

// prune drops the sessions that last reported a different window from the merged one, so
// the map stays small; a missing entry only makes a drop harder to accept.
func prune(sessions map[string]Limits, merged Limits) map[string]Limits {
	window := func(l Limits) *Reading { return l.FiveHour }
	if merged.FiveHour == nil {
		window = func(l Limits) *Reading { return l.SevenDay }
	}
	current := window(merged)
	for id, l := range sessions {
		if r := window(l); current == nil || r == nil || r.ResetsAt != current.ResetsAt {
			delete(sessions, id)
		}
	}
	if len(sessions) == 0 {
		return nil
	}
	return sessions
}

func (c *Cache) lock() func() {
	unlocked := func() {}
	if os.MkdirAll(filepath.Dir(c.path), 0o700) != nil {
		return unlocked
	}
	f, err := os.OpenFile(c.path+".lock", os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // G304: the user's own config directory
	if err != nil {
		return unlocked
	}
	deadline := time.Now().Add(lockWait)
	for syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil { //nolint:gosec // G115: a file descriptor fits in an int
		if time.Now().After(deadline) {
			_ = f.Close()
			return unlocked
		}
		time.Sleep(lockPoll)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:gosec // G115: a file descriptor fits in an int
		_ = f.Close()
	}
}

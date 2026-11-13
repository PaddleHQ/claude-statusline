// Package overage keeps a running estimate of extra-usage spend, shared between sessions. Claude Code reports no
// per-request flag for it, so the estimate counts the cost a session adds while an account window is at 100%.
package overage

import (
	"encoding/json"
	"maps"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/filelock"
)

const (
	// maxValue keeps costs within 12 digits, like every other number the status line accepts.
	maxValue    = 1e12
	monthLayout = "2006-01"
	sessionTTL  = 7 * 24 * time.Hour
	maxMonths   = 12
)

// PathIn is where the overage file lives inside a state directory.
func PathIn(stateDir string) string { return filepath.Join(stateDir, "cache", "overage") }

// Tracker is the file that shares the estimate between sessions.
type Tracker struct{ path string }

// New returns the Tracker stored at path; an empty path means no file.
func New(path string) *Tracker { return &Tracker{path: path} }

type month struct {
	USD float64 `json:"usd"`
}

type session struct {
	Cost float64 `json:"cost"`
	Over bool    `json:"over"`
	Seen int64   `json:"seen"`
}

type state struct {
	Months   map[string]month   `json:"months"`
	Sessions map[string]session `json:"sessions"`
}

// MonthKey is the calendar month, in UTC, that spend at the given time belongs to.
func MonthKey(now time.Time) string { return now.UTC().Format(monthLayout) }

// NextReset is the start of the month after the one containing now, in UTC: when extra-usage spend resets.
func NextReset(now time.Time) time.Time {
	y, m, _ := now.UTC().Date()
	return time.Date(y, m+1, 1, 0, 0, 0, 0, time.UTC)
}

// Month is the spend estimate for the UTC calendar month containing now, with that month's key; a missing or
// corrupt file counts as zero.
func (t *Tracker) Month(now time.Time) (key string, usd float64) {
	key = MonthKey(now)
	return key, t.read().Months[key].USD
}

// Record adds what one session's cost has risen by since its last reading, when both readings were over the limit.
// A reading that is not over the limit and follows none that was changes nothing, so the common case takes no lock
// and writes nothing. Failures are ignored: the estimate is a convenience.
func (t *Tracker) Record(sessionID string, cost float64, over bool, now time.Time) {
	if t.path == "" || sessionID == "" || !valid(cost) {
		return
	}
	if !over {
		if prev, ok := t.read().Sessions[sessionID]; !ok || !prev.Over {
			return
		}
	}
	defer filelock.Acquire(t.path + ".lock")()

	cached := t.read()
	next := cached.with(sessionID, session{Cost: cost, Over: over, Seen: now.Unix()}, MonthKey(now))
	prune(&next, now)
	if !reflect.DeepEqual(next, cached) {
		t.write(next)
	}
}

// with returns a copy of s holding the new reading of a session, the month's total raised by what the session's cost
// has risen by since the previous reading, when that was over the limit too.
func (s *state) with(id string, reading session, monthKey string) state {
	next := state{Months: maps.Clone(s.Months), Sessions: maps.Clone(s.Sessions)}
	if next.Months == nil {
		next.Months = map[string]month{}
	}
	if next.Sessions == nil {
		next.Sessions = map[string]session{}
	}
	if prev, ok := next.Sessions[id]; ok && prev.Over && reading.Cost > prev.Cost {
		m := next.Months[monthKey]
		m.USD = math.Min(m.USD+reading.Cost-prev.Cost, maxValue)
		next.Months[monthKey] = m
	}
	next.Sessions[id] = reading
	return next
}

func prune(s *state, now time.Time) {
	oldest := now.Add(-sessionTTL).Unix()
	maps.DeleteFunc(s.Sessions, func(_ string, v session) bool { return v.Seen < oldest })
	keys := slices.Sorted(maps.Keys(s.Months))
	for len(keys) > maxMonths {
		delete(s.Months, keys[0])
		keys = keys[1:]
	}
}

func valid(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) && n >= 0 && n <= maxValue }

// read loads the file; a missing or corrupt file, or a half entry, counts as empty.
func (t *Tracker) read() state {
	data, err := os.ReadFile(t.path) //nolint:gosec // G304: the path derives from the user's own Claude config directory
	if err != nil {
		return state{}
	}
	var raw struct {
		Months   map[string]json.RawMessage `json:"months"`
		Sessions map[string]json.RawMessage `json:"sessions"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return state{}
	}
	return state{Months: readMonths(raw.Months), Sessions: readSessions(raw.Sessions)}
}

func readMonths(entries map[string]json.RawMessage) map[string]month {
	var months map[string]month
	for key, entry := range entries {
		var m struct {
			USD *float64 `json:"usd"`
		}
		if _, err := time.Parse(monthLayout, key); err != nil || json.Unmarshal(entry, &m) != nil || m.USD == nil || !valid(*m.USD) {
			continue
		}
		if months == nil {
			months = map[string]month{}
		}
		months[key] = month{USD: *m.USD}
	}
	return months
}

func readSessions(entries map[string]json.RawMessage) map[string]session {
	var sessions map[string]session
	for id, entry := range entries {
		var e struct {
			Cost *float64 `json:"cost"`
			Over *bool    `json:"over"`
			Seen *int64   `json:"seen"`
		}
		if json.Unmarshal(entry, &e) != nil || e.Cost == nil || e.Over == nil || e.Seen == nil || !valid(*e.Cost) || *e.Seen < 0 {
			continue
		}
		if sessions == nil {
			sessions = map[string]session{}
		}
		sessions[id] = session{Cost: *e.Cost, Over: *e.Over, Seen: *e.Seen}
	}
	return sessions
}

// write replaces the file through a temporary file and a rename, so concurrent sessions never see a partial file.
func (t *Tracker) write(s state) {
	data, err := json.Marshal(s)
	if err != nil {
		return
	}
	dir := filepath.Dir(t.path)
	if os.MkdirAll(dir, 0o700) != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, ".overage-*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(append(data, '\n'))
	cerr := tmp.Close()
	if werr != nil || cerr != nil || os.Rename(tmp.Name(), t.path) != nil {
		_ = os.Remove(tmp.Name())
	}
}

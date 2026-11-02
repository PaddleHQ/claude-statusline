// Package cache provides the cache segment: how much of the latest request's input came from the prompt cache,
// or that the cache has expired since.
package cache

import (
	_ "embed"
	"fmt"
	"strconv"
	"time"
)

// Defaults is the text of the segment's defaults.yaml, for layering under the user's config.
//
//go:embed defaults.yaml
var Defaults string

const (
	name         = "cache"
	percentScale = 100
	cold         = "⧉ cold"
)

//go:generate go run github.com/vektra/mockery/v2 --name=Session

// Session is what the segment needs to know about the session.
type Session interface {
	// LatestRequest is the token usage of the latest request.
	LatestRequest() (input, cacheCreation, cacheRead int64)
	// TranscriptFile is the path of the session transcript, or "" when there is none.
	TranscriptFile() string
}

//go:generate go run github.com/vektra/mockery/v2 --name=Transcript

// Transcript is what the segment needs from the session transcript.
type Transcript interface {
	// LastRequest is when the latest request was answered and how long its prompt cache lives; at is zero when unknown.
	LastRequest() (at time.Time, ttl time.Duration)
}

//go:generate go run github.com/vektra/mockery/v2 --name=Config

// Config reads the segment's settings.
type Config interface {
	Int(key string) int
}

//go:generate go run github.com/vektra/mockery/v2 --name=Painter

// Painter colors text.
type Painter interface {
	Text(s string) string
	Bad(s string) string
}

// Deps is everything the segment is built from. Transcript is a function so that the transcript is only read when the segment needs it.
type Deps struct {
	Session    Session
	Transcript func() Transcript
	Config     Config
	Paint      Painter
	Now        time.Time
}

// Cache shows the prompt cache hit rate.
type Cache struct{ d Deps }

// New builds the cache segment.
func New(d Deps) *Cache { return &Cache{d: d} }

// Name is the segment's name in the segments list.
func (c *Cache) Name() string { return name }

// Help says what the segment shows, for the help command.
func (c *Cache) Help() string {
	return fmt.Sprintf("How much of the latest request's input was read from the prompt cache. Red below %d%%, "+
		"and a red %s once the cache has expired, as in a resumed session.", c.d.Config.Int("warn"), cold)
}

// Keys lists the config keys the segment reads, relative to its section.
func (c *Cache) Keys() []string { return []string{"warn"} }

// Render returns the segment's text, or "" to hide it.
func (c *Cache) Render() string {
	if c.expired() {
		return c.d.Paint.Bad(cold)
	}
	input, creation, read := c.d.Session.LatestRequest()
	total := input + creation + read
	if total <= 0 {
		return ""
	}
	pct := read * percentScale / total
	text := "⧉ " + strconv.FormatInt(pct, 10) + "%"
	if pct < int64(c.d.Config.Int("warn")) {
		return c.d.Paint.Bad(text)
	}
	return c.d.Paint.Text(text)
}

// expired reports whether the prompt cache has outlived its TTL since the last response, as it has in a resumed session.
func (c *Cache) expired() bool {
	if c.d.Session.TranscriptFile() == "" || c.d.Transcript == nil {
		return false
	}
	at, ttl := c.d.Transcript().LastRequest()
	return !at.IsZero() && !c.d.Now.Before(at.Add(ttl))
}

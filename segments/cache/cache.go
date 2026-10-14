// Package cache provides the cache segment: how much of the latest request's input came from the prompt cache.
package cache

import (
	"fmt"
	"strconv"
)

const (
	name         = "cache"
	percentScale = 100
)

//go:generate go run github.com/vektra/mockery/v2 --name=Session

// Session is what the segment needs to know about the session.
type Session interface {
	// LatestRequest is the token usage of the latest request.
	LatestRequest() (input, cacheCreation, cacheRead int64)
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

// Deps is everything the segment is built from.
type Deps struct {
	Session Session
	Config  Config
	Paint   Painter
}

// Cache shows the prompt cache hit rate.
type Cache struct{ d Deps }

// New builds the cache segment.
func New(d Deps) *Cache { return &Cache{d: d} }

// Name is the segment's name in the segments list.
func (c *Cache) Name() string { return name }

// Help says what the segment shows, for the help command.
func (c *Cache) Help() string {
	return fmt.Sprintf("How much of the latest request's input was read from the prompt cache. Red below %d%%.", c.d.Config.Int("warn"))
}

// Keys lists the config keys the segment reads, relative to its section.
func (c *Cache) Keys() []string { return []string{"warn"} }

// Render returns the segment's text, or "" to hide it.
func (c *Cache) Render() string {
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

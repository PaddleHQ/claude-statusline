// Package statusline turns the session payload and the environment into the printed line.
package statusline

import (
	"path/filepath"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/segment"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
)

const (
	keyTheme       = "theme"
	defaultColumns = 999
)

// Keys lists every config key the status line reads, for checking against defaults.conf.
func Keys() []string {
	return append(segment.Keys(), keyTheme)
}

// Build renders the line. It reads the user's config and the clock and width overrides via getenv.
func Build(input []byte, getenv func(string) string) string {
	cfg := config.New("")
	if path := config.Path(getenv); path != "" {
		cfg = config.Load(path)
	}

	th := theme.Lookup(cfg.String(keyTheme))
	if getenv("NO_COLOR") != "" {
		th = th.WithoutColor()
	}

	p := payload.Parse(input)
	at := now(getenv)
	rates := ratecache.Sync(cachePath(getenv), ratecache.FromPayload(p.RateLimits)).Visible(at.Unix())

	return segment.Render(&segment.Context{
		Payload: p,
		Rates:   rates,
		Config:  cfg,
		Theme:   th,
		Now:     at,
		Columns: columns(getenv),
		Home:    getenv("HOME"),
	})
}

func cachePath(getenv func(string) string) string {
	path := config.Path(getenv)
	if path == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(path), "cache", "ratelimits")
}

func now(getenv func(string) string) time.Time {
	if n, ok := config.ParseNumber(getenv("STATUSLINE_NOW")); ok {
		return time.Unix(int64(n), 0)
	}
	return time.Now()
}

func columns(getenv func(string) string) int {
	if n, ok := config.ParseNumber(getenv("COLUMNS")); ok {
		return n
	}
	return defaultColumns
}

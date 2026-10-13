// Package statusline turns the session payload and the environment into the printed line.
package statusline

import (
	"path/filepath"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/hook"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/segment"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
)

const (
	keyTheme       = "theme"
	keyHookTTL     = "hooks.ttl"
	keyHookTimeout = "hooks.timeout"
	defaultColumns = 999
)

// Keys lists every config key the status line reads, for checking against defaults.conf.
func Keys() []string {
	return append(segment.Keys(), keyTheme, keyHookTTL, keyHookTimeout)
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
		Hooks:   runHooks(input, p, cfg, at, getenv),
	})
}

// runHooks runs the scripts in the hook directory; STATUSLINE_HOOK_TTL and STATUSLINE_HOOK_TIMEOUT override the config.
func runHooks(input []byte, p payload.Payload, cfg *config.Config, at time.Time, getenv func(string) string) []hook.Result {
	configDir := config.Dir(getenv)
	dir := getenv("STATUSLINE_HOOK_DIR")
	if dir == "" && configDir != "" {
		dir = filepath.Join(configDir, "statusline.d")
	}
	hooks := hook.Discover(dir)
	if len(hooks) == 0 {
		return nil
	}
	opts := hook.Options{
		Input:      input,
		Session:    p.SessionID,
		Now:        at,
		Timeout:    seconds(getenv("STATUSLINE_HOOK_TIMEOUT"), cfg.Int(keyHookTimeout)),
		DefaultTTL: seconds(getenv("STATUSLINE_HOOK_TTL"), cfg.Int(keyHookTTL)),
	}
	if opts.Timeout <= 0 {
		opts.Timeout = time.Duration(cfg.Int(keyHookTimeout)) * time.Second
	}
	if configDir != "" {
		opts.CacheDir = filepath.Join(configDir, "claude-statusline", "cache", "hooks")
	}
	return hook.Run(hooks, opts)
}

func seconds(override string, fallback int) time.Duration {
	if n, ok := config.ParseNumber(override); ok {
		return time.Duration(n) * time.Second
	}
	return time.Duration(fallback) * time.Second
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

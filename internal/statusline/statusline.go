// Package statusline turns the session payload and the environment into the printed line.
package statusline

import (
	"path/filepath"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/render"
	"github.com/PaddleHQ/claude-statusline/internal/snapshot"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments"
)

const (
	keyTheme       = "theme"
	keySnapEnabled = "snapshot.enabled"
	keySnapPath    = "snapshot.path"
	defaultColumns = 999
)

// Keys lists every config key the status line reads, for checking against defaults.yaml.
func Keys() []string {
	keys := append(render.Keys(), keyTheme, keySnapEnabled, keySnapPath)
	for _, seg := range segments.NewEnv(segments.Env{Config: config.New("")}).All() {
		for _, k := range seg.Keys() {
			keys = append(keys, seg.Name()+"."+k)
		}
	}
	return keys
}

// Build renders the line. It reads the user's config and the clock and width overrides via getenv.
func Build(input []byte, getenv func(string) string) string {
	defaults := segments.Defaults()
	cfg := config.New("", defaults...)
	if path := config.Path(getenv); path != "" {
		cfg = config.Load(path, defaults...)
	}

	saveSnapshot(input, cfg, getenv)

	th := theme.Load(cfg.String(keyTheme), getenv("HOME"), config.AppDir(getenv))
	if getenv("NO_COLOR") != "" {
		th = th.WithoutColor()
	}

	p := payload.Parse(input)
	at := now(getenv)

	env := segments.NewEnv(segments.Env{
		Payload: p,
		Config:  cfg,
		Theme:   th,
		Now:     at,
		Columns: columns(getenv),
		Home:    getenv("HOME"),
		RateLimits: func() ratecache.Limits {
			synced := ratecache.NewCache(cachePath(getenv)).Sync(ratecache.FromPayload(p.RateLimits))
			return synced.Visible(at.Unix())
		},
	})
	line := &render.Line{
		Names:     cfg.Strings("segments"),
		Lookup:    lookup(env),
		Columns:   env.Columns,
		Separator: env.Separator(),
		Anchor:    th.Paint(th.Muted, "↳"),
	}
	return line.Render()
}

// lookup lets the renderer find the segments of env.
func lookup(env *segments.Env) func(string) (render.Segment, layout.Group, bool) {
	return func(name string) (render.Segment, layout.Group, bool) {
		seg, group, ok := env.Lookup(name)
		if !ok {
			return nil, 0, false
		}
		return seg, group, true
	}
}

// saveSnapshot keeps the latest payload where other tools can read it; a failed write never affects the line.
func saveSnapshot(input []byte, cfg *config.Config, getenv func(string) string) {
	if !cfg.Bool(keySnapEnabled) {
		return
	}
	path := cfg.String(keySnapPath)
	if path == "" {
		dir := config.Dir(getenv)
		if dir == "" {
			return
		}
		path = filepath.Join(dir, snapshot.FileName)
	}
	_ = snapshot.New(path).Write(input)
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

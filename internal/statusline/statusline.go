// Package statusline turns the session payload and the environment into the printed line.
package statusline

import (
	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/render"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments"
)

const (
	keyTheme       = "theme"
	defaultColumns = 999
)

// Keys lists every config key the status line reads, for checking against defaults.yaml.
func Keys() []string {
	keys := append(render.Keys(), keyTheme)
	for _, seg := range (&segments.Env{Config: config.New("")}).All() {
		for _, k := range seg.Keys() {
			keys = append(keys, seg.Name()+"."+k)
		}
	}
	return keys
}

// Build renders the line. It reads the user's config and the width override via getenv.
func Build(input []byte, getenv func(string) string) string {
	defaults := segments.Defaults()
	cfg := config.New("", defaults...)
	if path := config.Path(getenv); path != "" {
		cfg = config.Load(path, defaults...)
	}

	th := theme.Load(cfg.String(keyTheme), getenv("HOME"), config.AppDir(getenv))
	if getenv("NO_COLOR") != "" {
		th = th.WithoutColor()
	}

	env := &segments.Env{
		Payload: payload.Parse(input),
		Config:  cfg,
		Theme:   th,
		Columns: columns(getenv),
		Home:    getenv("HOME"),
	}
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

func columns(getenv func(string) string) int {
	if n, ok := config.ParseNumber(getenv("COLUMNS")); ok {
		return n
	}
	return defaultColumns
}

// Package statusline turns the session payload and the environment into the printed line.
package statusline

import (
	"path/filepath"
	"slices"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/hook"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/ratecache"
	"github.com/PaddleHQ/claude-statusline/internal/render"
	"github.com/PaddleHQ/claude-statusline/internal/snapshot"
	"github.com/PaddleHQ/claude-statusline/internal/subagent"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/internal/transcript"
	"github.com/PaddleHQ/claude-statusline/segments"
)

const (
	keySegments    = "segments"
	keyExcluded    = "segments_excluded"
	keyTheme       = "theme"
	keyHookTTL     = "hooks.ttl"
	keyHookTimeout = "hooks.timeout"
	keySnapEnabled = "snapshot.enabled"
	keySnapPath    = "snapshot.path"
	defaultColumns = 999
)

// Keys lists every config key the status line reads, for checking against defaults.yaml.
func Keys() []string {
	keys := append(render.Keys(), keyExcluded, keyTheme, keyHookTTL, keyHookTimeout, keySnapEnabled, keySnapPath)
	for _, seg := range segments.NewEnv(segments.Env{Config: config.New("")}).All() {
		for _, k := range seg.Keys() {
			keys = append(keys, seg.Name()+"."+k)
		}
	}
	return append(keys, subagent.Keys()...)
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
		Transcript: func() transcript.Info {
			return transcript.Read(p.TranscriptPath)
		},
		RateLimits: func() ratecache.Limits {
			synced := ratecache.NewCache(cachePath(getenv)).Sync(ratecache.FromPayload(p.RateLimits))
			return synced.Visible(at.Unix())
		},
	})
	hooks := hook.Discover(hookDir(getenv))
	fromList, fromHooks := exclusions(cfg, hooks)
	line := &render.Line{
		Names:     segmentNames(cfg, fromList),
		Lookup:    lookup(env),
		Columns:   env.Columns,
		Separator: env.Separator(),
		Anchor:    th.Paint(th.Muted, "↳"),
		Hooks:     runHooks(withoutExcluded(hooks, fromHooks), input, p, cfg, at, getenv),
	}
	return withErrorRow(line.Render(), cfg.Problem(), &th, getenv("HOME"), env.Columns)
}

// withErrorRow puts the config error above the line, or shows it alone when the line is empty.
func withErrorRow(line string, p *config.Problem, th *theme.Theme, home string, columns int) string {
	if p == nil {
		return line
	}
	row := errorRow(p, th, home, columns)
	if line == "" {
		return row
	}
	return row + "\n" + line
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
	} else if !filepath.IsAbs(path) {
		// The working directory is the repository Claude Code runs in, not a place for the user's own files.
		dir := config.AppDir(getenv)
		if dir == "" {
			return
		}
		path = filepath.Join(dir, path)
	}
	_ = snapshot.New(path).Write(input)
}

// runHooks runs the hook scripts; STATUSLINE_HOOK_TTL and STATUSLINE_HOOK_TIMEOUT override the config.
func runHooks(
	hooks []hook.Hook, input []byte, p payload.Payload, cfg *config.Config, at time.Time, getenv func(string) string,
) []hook.Result {
	if len(hooks) == 0 {
		return nil
	}
	opts := &hook.Options{
		Input:      input,
		Session:    p.SessionID,
		Now:        at,
		Timeout:    seconds(getenv("STATUSLINE_HOOK_TIMEOUT"), cfg.Int(keyHookTimeout)),
		DefaultTTL: seconds(getenv("STATUSLINE_HOOK_TTL"), cfg.Int(keyHookTTL)),
	}
	if opts.Timeout <= 0 {
		opts.Timeout = time.Duration(cfg.Int(keyHookTimeout)) * time.Second
	}
	if stateDir := config.StateDir(getenv); stateDir != "" {
		opts.CacheDir = filepath.Join(stateDir, "cache", "hooks")
	}
	return opts.Run(hooks)
}

// exclusions splits segments_excluded into the names to leave out of the segments list and the hooks not to run.
// The default only turns off built-ins: a user's own script of the same name is placed as if nothing were excluded,
// in its slot under the default segments list, or where the user's own list names it.
func exclusions(cfg *config.Config, hooks []hook.Hook) (fromList, fromHooks []string) {
	excluded := cfg.StringsOrNone(keyExcluded)
	if cfg.HasUserList(keyExcluded) || len(excluded) == 0 {
		return excluded, excluded
	}
	if !cfg.HasUserList(keySegments) {
		return excluded, nil
	}
	return slices.DeleteFunc(excluded, func(name string) bool {
		return slices.ContainsFunc(hooks, func(h hook.Hook) bool { return h.Name == name })
	}), nil
}

// segmentNames is the segments list less the excluded names.
func segmentNames(cfg *config.Config, excluded []string) []string {
	return slices.DeleteFunc(cfg.Strings(keySegments), func(name string) bool { return slices.Contains(excluded, name) })
}

// withoutExcluded drops excluded hooks, which would otherwise fall back to their slot once their name left the segments list.
func withoutExcluded(hooks []hook.Hook, excluded []string) []hook.Hook {
	return slices.DeleteFunc(hooks, func(h hook.Hook) bool { return slices.Contains(excluded, h.Name) })
}

func hookDir(getenv func(string) string) string {
	// A relative path would resolve against the repository Claude Code is running in, which the user does not control.
	if dir := getenv("STATUSLINE_HOOK_DIR"); filepath.IsAbs(dir) {
		return dir
	}
	if configDir := config.Dir(getenv); configDir != "" {
		return filepath.Join(configDir, "statusline.d")
	}
	return ""
}

func seconds(override string, fallback int) time.Duration {
	if n, ok := config.ParseNumber(override); ok {
		return time.Duration(n) * time.Second
	}
	return time.Duration(fallback) * time.Second
}

func cachePath(getenv func(string) string) string {
	dir := config.StateDir(getenv)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "cache", "ratelimits")
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

package statusline_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/statusline"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestKeysMatchDefaults(t *testing.T) {
	t.Parallel()

	used := slices.Compact(slices.Sorted(slices.Values(statusline.Keys())))
	defined := slices.Sorted(slices.Values(config.DefaultKeys()))

	for _, k := range used {
		assert.Contains(t, defined, k, "key %q is read by the code but missing from defaults.conf", k)
	}
	for _, k := range defined {
		assert.Contains(t, used, k, "key %q is in defaults.conf but nothing reads it", k)
	}
}

func TestNow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
		want int64
	}{
		{"uses STATUSLINE_NOW", map[string]string{"STATUSLINE_NOW": "1760000000"}, 1760000000},
		{"strips leading zeros", map[string]string{"STATUSLINE_NOW": "0000000012"}, 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, statusline.Now(env(tt.env)).Unix())
		})
	}

	for _, bad := range []string{"", "abc", "-1", "1e9", "1760000000000000"} {
		before := time.Now().Add(-time.Minute)
		got := statusline.Now(env(map[string]string{"STATUSLINE_NOW": bad}))
		assert.False(t, got.Before(before) || got.After(time.Now().Add(time.Minute)), "now() with STATUSLINE_NOW=%q = %v, want the current time", bad, got)
	}
}

func TestColumns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value string
		want  int
	}{
		{"100", 100},
		{"", 999},
		{"abc", 999},
		{"-5", 999},
		{"12x", 999},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, statusline.Columns(env(map[string]string{"COLUMNS": tt.value})), "COLUMNS=%q", tt.value)
	}
}

func TestBuild(t *testing.T) {
	t.Parallel()

	const input = `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"}}`

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "claude-statusline"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "claude-statusline", "config"), []byte("model.effort = false\n"), 0o600))

	tests := []struct {
		name  string
		input string
		env   map[string]string
		want  string
	}{
		{"defaults when no config exists", input, map[string]string{"CLAUDE_CONFIG_DIR": t.TempDir()}, "Opus 4.7 [high]"},
		{"reads the user's config", input, map[string]string{"CLAUDE_CONFIG_DIR": dir}, "Opus 4.7"},
		{"no config directory at all", input, map[string]string{}, "Opus 4.7 [high]"},
		{"invalid input", `{"model":`, map[string]string{"CLAUDE_CONFIG_DIR": dir}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, statusline.Build([]byte(tt.input), env(tt.env)))
		})
	}
}

func writeHook(t *testing.T, dir, name, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body+"\n"), 0o700))
}

func TestBuildHooks(t *testing.T) {
	t.Parallel()

	const input = `{"model":{"display_name":"Opus"},"session_id":"sess1"}`

	t.Run("runs the hooks in the default directory with the session JSON", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeHook(t, filepath.Join(dir, "statusline.d"), "10-echo.sh", `grep -o sess1`)
		got := statusline.Build([]byte(input), env(map[string]string{"CLAUDE_CONFIG_DIR": dir, "NO_COLOR": "1"}))
		assert.Equal(t, "Opus · sess1", got)
		assert.FileExists(t, filepath.Join(dir, "claude-statusline", "cache", "hooks", "sess1", "10-echo.sh"))
	})

	t.Run("STATUSLINE_HOOK_DIR overrides the directory", func(t *testing.T) {
		t.Parallel()
		hooks := t.TempDir()
		writeHook(t, hooks, "10-x.sh", `printf custom`)
		got := statusline.Build([]byte(input), env(map[string]string{"CLAUDE_CONFIG_DIR": t.TempDir(), "STATUSLINE_HOOK_DIR": hooks, "NO_COLOR": "1"}))
		assert.Equal(t, "Opus · custom", got)
	})

	t.Run("STATUSLINE_HOOK_TTL overrides the config", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeHook(t, filepath.Join(dir, "statusline.d"), "10-x.sh", `printf fresh`)
		cache := filepath.Join(dir, "claude-statusline", "cache", "hooks", "sess1")
		require.NoError(t, os.MkdirAll(cache, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(cache, "10-x.sh"), []byte("1759999000\nstale\n"), 0o600))
		base := map[string]string{"CLAUDE_CONFIG_DIR": dir, "NO_COLOR": "1", "STATUSLINE_NOW": "1760000000"}

		assert.Equal(t, "Opus · fresh", statusline.Build([]byte(input), env(base)), "1000 seconds old is past the default 60")
		require.NoError(t, os.WriteFile(filepath.Join(cache, "10-x.sh"), []byte("1759999000\nstale\n"), 0o600))
		base["STATUSLINE_HOOK_TTL"] = "5000"
		assert.Equal(t, "Opus · stale", statusline.Build([]byte(input), env(base)))
	})

	t.Run("STATUSLINE_HOOK_TIMEOUT overrides the config", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeHook(t, filepath.Join(dir, "statusline.d"), "10-slow.sh", `sleep 5; printf late`)
		start := time.Now()
		got := statusline.Build([]byte(input), env(map[string]string{"CLAUDE_CONFIG_DIR": dir, "NO_COLOR": "1", "STATUSLINE_HOOK_TIMEOUT": "1"}))
		assert.Equal(t, "Opus", got)
		assert.Less(t, time.Since(start), 3*time.Second)
	})

	t.Run("hooks.timeout from the config applies", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeHook(t, filepath.Join(dir, "statusline.d"), "10-slow.sh", `sleep 5; printf late`)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "claude-statusline"), 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "claude-statusline", "config"), []byte("hooks.timeout = 1\n"), 0o600))
		start := time.Now()
		got := statusline.Build([]byte(input), env(map[string]string{"CLAUDE_CONFIG_DIR": dir, "NO_COLOR": "1"}))
		assert.Equal(t, "Opus", got)
		assert.Less(t, time.Since(start), 3*time.Second)
	})
}

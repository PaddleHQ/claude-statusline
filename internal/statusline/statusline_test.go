package statusline_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/statusline"
	"github.com/PaddleHQ/claude-statusline/segments"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestKeysMatchDefaults(t *testing.T) {
	t.Parallel()

	used := slices.Compact(slices.Sorted(slices.Values(statusline.Keys())))
	defined := slices.Sorted(slices.Values(config.DefaultKeys(segments.Defaults()...)))

	for _, k := range used {
		assert.Contains(t, defined, k, "key %q is read by the code but missing from the defaults", k)
	}
	for _, k := range defined {
		assert.Contains(t, used, k, "key %q is in the defaults but nothing reads it", k)
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

func TestBuildSnapshot(t *testing.T) {
	t.Parallel()

	const input = "{\"model\":{\"display_name\":\"Opus\"},\n \"rate_limits\":{\"five_hour\":{\"used_percentage\":12}}}\n"

	configure := func(t *testing.T, dir, text string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "claude-statusline"), 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "claude-statusline", "config"), []byte(text), 0o600))
	}

	t.Run("writes the payload unchanged with mode 0600", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		got := statusline.Build([]byte(input), env(map[string]string{"CLAUDE_CONFIG_DIR": dir, "NO_COLOR": "1"}))

		assert.Equal(t, "Opus", got)
		path := filepath.Join(dir, "usage-latest.json")
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.JSONEq(t, input, string(data))
		fi, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
	})

	t.Run("falls back to HOME/.claude", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()

		statusline.Build([]byte(input), env(map[string]string{"HOME": home}))

		assert.FileExists(t, filepath.Join(home, ".claude", "usage-latest.json"))
	})

	t.Run("honors snapshot.path", func(t *testing.T) {
		t.Parallel()
		dir, elsewhere := t.TempDir(), filepath.Join(t.TempDir(), "out", "latest.json")
		configure(t, dir, "snapshot.path = "+elsewhere+"\n")

		statusline.Build([]byte(input), env(map[string]string{"CLAUDE_CONFIG_DIR": dir}))

		assert.FileExists(t, elsewhere)
		assert.NoFileExists(t, filepath.Join(dir, "usage-latest.json"))
	})

	t.Run("is disabled by snapshot.enabled", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		configure(t, dir, "snapshot.enabled = false\n")

		statusline.Build([]byte(input), env(map[string]string{"CLAUDE_CONFIG_DIR": dir}))

		assert.NoFileExists(t, filepath.Join(dir, "usage-latest.json"))
	})

	t.Run("does not write input that is not valid JSON", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		for _, bad := range []string{`{"model":`, ``, "not json"} {
			statusline.Build([]byte(bad), env(map[string]string{"CLAUDE_CONFIG_DIR": dir}))
		}

		assert.NoFileExists(t, filepath.Join(dir, "usage-latest.json"))
	})

	t.Run("keeps the previous snapshot when a later payload is invalid", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		e := env(map[string]string{"CLAUDE_CONFIG_DIR": dir})

		statusline.Build([]byte(input), e)
		statusline.Build([]byte(`{"model":`), e)

		data, err := os.ReadFile(filepath.Join(dir, "usage-latest.json"))
		require.NoError(t, err)
		assert.JSONEq(t, input, string(data))
	})

	t.Run("an unwritable directory does not break rendering", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		blocker := filepath.Join(dir, "file")
		require.NoError(t, os.WriteFile(blocker, nil, 0o600))
		configure(t, dir, "snapshot.path = "+filepath.Join(blocker, "sub", "latest.json")+"\n")

		got := statusline.Build([]byte(input), env(map[string]string{"CLAUDE_CONFIG_DIR": dir, "NO_COLOR": "1"}))

		assert.Equal(t, "Opus", got)
	})

	t.Run("leaves no temporary files behind", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		statusline.Build([]byte(input), env(map[string]string{"CLAUDE_CONFIG_DIR": dir}))

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		for _, e := range entries {
			assert.False(t, strings.HasPrefix(e.Name(), ".usage-"), "left %s behind", e.Name())
		}
	})
}

// Not parallel: t.Chdir changes the working directory of the whole process.
func TestBuildIgnoresRelativeHookDir(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(tmp, "rel"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "rel", "x.sh"), []byte("#!/bin/sh\nprintf PWNED\n"), 0o700)) //nolint:gosec // test hook must be executable
	t.Chdir(tmp)

	got := statusline.Build([]byte(`{"session_id":"s1"}`), env(map[string]string{"CLAUDE_CONFIG_DIR": t.TempDir(), "STATUSLINE_HOOK_DIR": "rel", "NO_COLOR": "1"}))

	assert.NotContains(t, got, "PWNED")
}

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Each testdata/golden/<case>/ holds input.json and expected, plus optional config, env, cache (seeded
// before the run), tree (files created in the temp directory before the run) and expected_cache (compared after it).
// Each tree line is path=content, relative to the temp directory; a path ending in / makes a directory.
// Escape bytes in the output are compared as the text \e, and @TMP@ stands for the case's temp directory
// in input.json, config, tree and expected.
func TestGolden(t *testing.T) {
	t.Parallel()

	cases, err := filepath.Glob(filepath.Join("testdata", "golden", "*"))
	require.NoError(t, err)
	require.NotEmpty(t, cases, "no golden cases found")

	for _, dir := range cases {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			t.Parallel()
			runGolden(t, dir)
		})
	}
}

func runGolden(t *testing.T, dir string) {
	t.Helper()

	tmp := t.TempDir()
	env := map[string]string{
		"HOME":              tmp,
		"CLAUDE_CONFIG_DIR": tmp,
		"COLUMNS":           "200",
		"STATUSLINE_NOW":    "1760000000",
	}
	if data, err := os.ReadFile(filepath.Join(dir, "env")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if k, v, ok := strings.Cut(line, "="); ok && !strings.HasPrefix(k, "#") {
				env[k] = v
			}
		}
	}

	if data, err := os.ReadFile(filepath.Join(dir, "config")); err == nil {
		appDir := filepath.Join(tmp, "claude-statusline")
		require.NoError(t, os.MkdirAll(appDir, 0o700))
		text := strings.ReplaceAll(string(data), "@TMP@", tmp)
		require.NoError(t, os.WriteFile(filepath.Join(appDir, "config"), []byte(text), 0o600))
	}

	cacheFile := filepath.Join(tmp, "claude-statusline", "cache", "ratelimits")
	if data, err := os.ReadFile(filepath.Join(dir, "cache")); err == nil {
		require.NoError(t, os.MkdirAll(filepath.Dir(cacheFile), 0o700))
		require.NoError(t, os.WriteFile(cacheFile, data, 0o600))
	}

	if data, err := os.ReadFile(filepath.Join(dir, "tree")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if line == "" {
				continue
			}
			rel, content, _ := strings.Cut(line, "=")
			path := filepath.Join(tmp, rel)
			if strings.HasSuffix(rel, "/") {
				require.NoError(t, os.MkdirAll(path, 0o700))
				continue
			}
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
			require.NoError(t, os.WriteFile(path, []byte(strings.ReplaceAll(content, "@TMP@", tmp)+"\n"), 0o600))
		}
	}

	input, err := os.ReadFile(filepath.Join(dir, "input.json"))
	require.NoError(t, err)
	input = []byte(strings.ReplaceAll(string(input), "@TMP@", tmp))
	want, err := os.ReadFile(filepath.Join(dir, "expected"))
	require.NoError(t, err)

	var out bytes.Buffer
	run(bytes.NewReader(input), &out, func(k string) string { return env[k] })

	got := strings.ReplaceAll(out.String(), "\033", `\e`)
	wantText := strings.TrimSuffix(strings.ReplaceAll(string(want), "@TMP@", tmp), "\n")
	assert.Equal(t, wantText, got)
	assert.NotContains(t, out.String(), "\n", "output contains a newline")
	if wantCache, err := os.ReadFile(filepath.Join(dir, "expected_cache")); err == nil {
		gotCache, _ := os.ReadFile(cacheFile)
		assert.Equal(t, strings.TrimSpace(string(wantCache)), strings.TrimSpace(string(gotCache)), "cache mismatch")
	}
	assert.NoFileExists(t, filepath.Join(tmp, "pwned"), "a config value was executed")
}

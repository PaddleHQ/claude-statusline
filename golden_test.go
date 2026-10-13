package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Each testdata/golden/<case>/ holds input.json and expected, plus optional config, env, cache (seeded
// before the run) and expected_cache (compared after it).
// Escape bytes in the output are compared as the text \e, and @TMP@ stands for the case's temp directory.
func TestGolden(t *testing.T) {
	t.Parallel()

	cases, err := filepath.Glob(filepath.Join("testdata", "golden", "*"))
	if err != nil || len(cases) == 0 {
		t.Fatalf("no golden cases found: %v", err)
	}

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
		if err := os.MkdirAll(appDir, 0o700); err != nil {
			t.Fatal(err)
		}
		text := strings.ReplaceAll(string(data), "@TMP@", tmp)
		if err := os.WriteFile(filepath.Join(appDir, "config"), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cacheFile := filepath.Join(tmp, "claude-statusline", "cache", "ratelimits")
	if data, err := os.ReadFile(filepath.Join(dir, "cache")); err == nil {
		if err := os.MkdirAll(filepath.Dir(cacheFile), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cacheFile, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	input, err := os.ReadFile(filepath.Join(dir, "input.json"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(dir, "expected"))
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	run(bytes.NewReader(input), &out, func(k string) string { return env[k] })

	got := strings.ReplaceAll(out.String(), "\033", `\e`)
	wantText := strings.TrimSuffix(strings.ReplaceAll(string(want), "@TMP@", tmp), "\n")
	if got != wantText {
		t.Errorf("output mismatch\n got: %s\nwant: %s", got, wantText)
	}
	if strings.Contains(out.String(), "\n") {
		t.Error("output contains a newline")
	}
	if wantCache, err := os.ReadFile(filepath.Join(dir, "expected_cache")); err == nil {
		gotCache, _ := os.ReadFile(cacheFile)
		if strings.TrimSpace(string(gotCache)) != strings.TrimSpace(string(wantCache)) {
			t.Errorf("cache mismatch\n got: %s\nwant: %s", gotCache, wantCache)
		}
	}
	if _, err := os.Stat(filepath.Join(tmp, "pwned")); err == nil {
		t.Error("a config value was executed")
	}
}

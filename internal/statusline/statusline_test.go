package statusline

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/config"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestKeysMatchDefaults(t *testing.T) {
	t.Parallel()

	used := slices.Compact(slices.Sorted(slices.Values(Keys())))
	defined := slices.Sorted(slices.Values(config.DefaultKeys()))

	for _, k := range used {
		if !slices.Contains(defined, k) {
			t.Errorf("key %q is read by the code but missing from defaults.conf", k)
		}
	}
	for _, k := range defined {
		if !slices.Contains(used, k) {
			t.Errorf("key %q is in defaults.conf but nothing reads it", k)
		}
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
			if got := now(env(tt.env)).Unix(); got != tt.want {
				t.Errorf("now() = %d, want %d", got, tt.want)
			}
		})
	}

	for _, bad := range []string{"", "abc", "-1", "1e9", "1760000000000000"} {
		before := time.Now().Add(-time.Minute)
		if got := now(env(map[string]string{"STATUSLINE_NOW": bad})); got.Before(before) || got.After(time.Now().Add(time.Minute)) {
			t.Errorf("now() with STATUSLINE_NOW=%q = %v, want the current time", bad, got)
		}
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
		if got := columns(env(map[string]string{"COLUMNS": tt.value})); got != tt.want {
			t.Errorf("columns(%q) = %d, want %d", tt.value, got, tt.want)
		}
	}
}

func TestBuild(t *testing.T) {
	t.Parallel()

	const input = `{"model":{"display_name":"Opus 4.7"},"effort":{"level":"high"}}`

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "claude-statusline"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "claude-statusline", "config"), []byte("model.effort = false\n"), 0o600); err != nil {
		t.Fatal(err)
	}

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
			if got := Build([]byte(tt.input), env(tt.env)); got != tt.want {
				t.Errorf("Build() = %q, want %q", got, tt.want)
			}
		})
	}
}

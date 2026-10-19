package statusline_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

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

func TestDefaultsAreDocumented(t *testing.T) {
	t.Parallel()

	for _, text := range append([]string{config.Defaults}, segments.Defaults()...) {
		assert.Empty(t, config.UncommentedKeys(text), "keys in a defaults.yaml need a comment on the line above")
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
	require.NoError(t, os.WriteFile(filepath.Join(dir, "claude-statusline", "config.yaml"), []byte("model:\n  effort: false\n"), 0o600))

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

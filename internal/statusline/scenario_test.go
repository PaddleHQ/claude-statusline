package statusline_test

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/statusline"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tmpMarker = "@TMP@"

// scenario is one run of the command in a private home directory. The text of every field may name that directory
// as @TMP@, and the expected output writes an escape byte as \e.
type scenario struct {
	name        string
	session     string
	config      string
	env         map[string]string
	files       map[string]string
	executables map[string]string
	cache       string
	want        string
	wantCache   string
}

// home creates the scenario's directory with its config, files and seeded cache, and returns it with the environment.
func (s *scenario) home(t *testing.T) (dir string, getenv func(string) string) {
	t.Helper()

	dir = t.TempDir()
	vars := map[string]string{
		"HOME":                    dir,
		"CLAUDE_CONFIG_DIR":       dir,
		"STATUSLINE_HOOK_TIMEOUT": "30",
		"STATUSLINE_NOW":          "1760000000",
		"COLUMNS":                 "999",
	}
	maps.Copy(vars, s.env)
	write := func(rel, text string, perm os.FileMode) {
		path := filepath.Join(dir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(strings.ReplaceAll(text, tmpMarker, dir)+"\n"), perm))
	}
	if s.config != "" {
		write(filepath.Join("claude-statusline", "config.yaml"), s.config, 0o600)
	}
	if s.cache != "" {
		write(filepath.Join("claude-statusline", "cache", "ratelimits"), s.cache, 0o600)
	}
	for rel, text := range s.files {
		write(rel, text, 0o600)
	}
	for rel, text := range s.executables {
		write(rel, text, 0o700)
	}
	return dir, func(k string) string { return vars[k] }
}

func (s *scenario) expected(dir string) string { return strings.ReplaceAll(s.want, tmpMarker, dir) }

func escaped(text string) string { return strings.ReplaceAll(text, "\x1b", `\e`) }

// build runs the status line for the scenario and checks the line, and the shared cache when the scenario names it.
func (s *scenario) build(t *testing.T) {
	t.Helper()

	dir, getenv := s.home(t)
	got := statusline.Build([]byte(strings.ReplaceAll(s.session, tmpMarker, dir)), getenv)

	assert.Equal(t, s.expected(dir), escaped(got))
	assert.NoFileExists(t, filepath.Join(dir, "pwned"), "a config value was executed")
	if s.wantCache != "" {
		data, err := os.ReadFile(filepath.Join(dir, "claude-statusline", "cache", "ratelimits"))
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(s.wantCache), strings.TrimSpace(string(data)))
	}
}

func runAll(t *testing.T, tests []scenario) {
	t.Helper()

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.build(t)
		})
	}
}

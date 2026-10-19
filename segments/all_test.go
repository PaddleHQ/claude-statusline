package segments_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func newEnv(cfg, payloadJSON string) *segments.Env {
	return &segments.Env{
		Payload: payload.Parse([]byte(payloadJSON)),
		Config:  config.New(cfg, segments.Defaults()...),
		Theme:   theme.Load("classic", "", ""),
		Columns: 999,
	}
}

func TestEveryRegisteredNameIsUniqueAndFindable(t *testing.T) {
	t.Parallel()

	env := newEnv("", `{}`)
	seen := map[string]bool{}
	for _, seg := range env.All() {
		assert.False(t, seen[seg.Name()], "%q is provided twice", seg.Name())
		seen[seg.Name()] = true
		found, _, ok := env.Lookup(seg.Name())
		require.True(t, ok, seg.Name())
		assert.Equal(t, seg.Name(), found.Name())
	}
	assert.Len(t, seen, 1)
	_, _, ok := env.Lookup("bogus")
	assert.False(t, ok)
}

func TestLayoutGroups(t *testing.T) {
	t.Parallel()

	want := map[string]layout.Group{
		"model": layout.Identity,
	}
	env := newEnv("", `{}`)
	for name, group := range want {
		_, got, ok := env.Lookup(name)
		require.True(t, ok, name)
		assert.Equal(t, group, got, name)
	}
}

func pluginDirs(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	require.NotEmpty(t, dirs)
	return dirs
}

func TestPluginsImportNothingFromInternal(t *testing.T) {
	t.Parallel()

	for _, dir := range pluginDirs(t) {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, perr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			require.NoError(t, perr, path)
			for _, imp := range file.Imports {
				assert.NotContains(t, imp.Path.Value, "/internal/", "%s imports %s", path, imp.Path.Value)
			}
			return nil
		})
		require.NoError(t, err)
	}
}

func TestEveryPluginReadmeDocumentsItsDefaults(t *testing.T) {
	t.Parallel()

	for _, dir := range pluginDirs(t) {
		readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
		require.NoError(t, err, "%s needs a README.md", dir)
		defaults, err := os.ReadFile(filepath.Join(dir, "defaults.yaml"))
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)
		var sections map[string]map[string]any
		require.NoError(t, yaml.Unmarshal(defaults, &sections), dir)
		for section, keys := range sections {
			for key, value := range keys {
				row := fmt.Sprintf("| `%s.%s` | %#q |", section, key, readmeValue(value))
				assert.Contains(t, string(readme), row, "%s/README.md must list %s.%s with its default", dir, section, key)
			}
		}
	}
}

func readmeValue(v any) string {
	list, ok := v.([]any)
	if !ok {
		return fmt.Sprint(v)
	}
	items := make([]string, 0, len(list))
	for _, item := range list {
		items = append(items, fmt.Sprint(item))
	}
	return "[" + strings.Join(items, ", ") + "]"
}

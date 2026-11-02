package statusline_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/statusline"
	"github.com/PaddleHQ/claude-statusline/segments"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type userConfigField struct {
	Type     string   `json:"type"`
	Title    string   `json:"title"`
	Multiple bool     `json:"multiple"`
	Default  any      `json:"default"`
	Options  []string `json:"options"`
	Required bool     `json:"required"`
}

func TestPluginManifestDeclaresEveryOption(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../../.claude-plugin/plugin.json")
	require.NoError(t, err)
	var manifest struct {
		Name       string                     `json:"name"`
		UserConfig map[string]userConfigField `json:"userConfig"`
	}
	require.NoError(t, json.Unmarshal(data, &manifest))

	assert.Equal(t, config.PluginName, manifest.Name)
	known := statusline.Keys()
	defaults := config.DefaultKeys(segments.Defaults()...)
	keyName := regexp.MustCompile(`^[A-Za-z_]\w*$`)

	declared := make([]string, 0, len(manifest.UserConfig))
	for _, o := range config.Options() {
		field, ok := manifest.UserConfig[o.Name]
		require.True(t, ok, "plugin.json does not declare %s", o.Name)
		declared = append(declared, o.Name)

		assert.Regexp(t, keyName, o.Name)
		assert.Contains(t, known, o.Key, "%s maps to a key the status line does not read", o.Name)
		assert.Contains(t, defaults, o.Key, "%s maps to a key with no default", o.Name)
		assert.NotEmpty(t, field.Title)
		if o.Name == "theme_file" {
			assert.Equal(t, "file", field.Type)
		} else {
			assert.Equal(t, "string", field.Type)
		}
		assert.False(t, field.Required)
		switch o.Kind {
		case config.KindList:
			assert.True(t, field.Multiple, o.Name)
			assert.Equal(t, []any{}, field.Default, o.Name)
		case config.KindToggle:
			assert.Equal(t, []string{"default", "on", "off"}, field.Options, o.Name)
			assert.Equal(t, "default", field.Default, o.Name)
		case config.KindChoice:
			assert.Equal(t, append([]string{"default"}, builtinThemes(t)...), field.Options, o.Name)
			assert.Equal(t, "default", field.Default, o.Name)
		case config.KindString:
			assert.False(t, field.Multiple, o.Name)
			assert.Empty(t, field.Options, o.Name)
			assert.Empty(t, field.Default, o.Name)
		}
	}
	for name := range manifest.UserConfig {
		assert.True(t, slices.Contains(declared, name), "%s is declared in plugin.json but maps to no config key", name)
	}
}

func builtinThemes(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob("../../themes/*.yaml")
	require.NoError(t, err)
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, strings.TrimSuffix(filepath.Base(f), ".yaml"))
	}
	require.NotEmpty(t, names)
	sort.Strings(names)
	return names
}

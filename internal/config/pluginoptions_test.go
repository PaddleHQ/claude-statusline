package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const entry = "statusline@claude-statusline"

func settings(options string) string {
	return fmt.Sprintf(`{"enabledPlugins": {%q: true}, "pluginConfigs": {%q: {"options": %s}}}`, entry, entry, options)
}

// resolve writes the two files into a fresh config directory and resolves the config from it.
func resolve(t *testing.T, settingsText, configText string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	if settingsText != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "settings.json"), []byte(settingsText), 0o600))
	}
	if configText != "" {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "claude-statusline"), 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "claude-statusline", "config.yaml"), []byte(configText), 0o600))
	}
	env := map[string]string{"CLAUDE_CONFIG_DIR": dir, "HOME": dir}
	return config.Resolve(func(k string) string { return env[k] }, segmentDefaults)
}

const segmentDefaults = `
path: {strip: [], branch: true, worktrees: [], emoji: false}
model: {effort: true}
`

func TestResolveOptionPrecedence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		options    string
		file       string
		wantString string
		wantSource string
	}{
		{"option wins over file", `{"theme": "saber"}`, "theme: gradient\n", "saber", config.SourcePlugin},
		{"option wins over default", `{"theme": "saber"}`, "", "saber", config.SourcePlugin},
		{"empty option falls through to file", `{"theme": ""}`, "theme: gradient\n", "gradient", config.SourceFile},
		{"empty option falls through to default", `{"theme": ""}`, "", "classic", config.SourceDefaults},
		{"missing option falls through to file", `{}`, "theme: gradient\n", "gradient", config.SourceFile},
		{"null option falls through", `{"theme": null}`, "", "classic", config.SourceDefaults},
		{"file only", `{}`, "theme: gradient\n", "gradient", config.SourceFile},
		{"neither", `{}`, "", "classic", config.SourceDefaults},
		{"wrong type is ignored like the same value in the file", `{"theme": 7}`, "theme: gradient\n", "classic", config.SourcePlugin},
		{"the word default is a theme name, not a fall-through", `{"theme": "default"}`, "theme: gradient\n", "default", config.SourcePlugin},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := resolve(t, settings(tt.options), tt.file)

			assert.Equal(t, tt.wantString, cfg.String("theme"))
			assert.Equal(t, tt.wantSource, cfg.Source("theme"))
		})
	}
}

func TestResolveEveryKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options string
		check   func(t *testing.T, cfg *config.Config)
	}{
		{"segments", `{"segments": ["model", "path"]}`, func(t *testing.T, c *config.Config) {
			assert.Equal(t, []string{"model", "path"}, c.Strings("segments"))
			assert.Equal(t, config.SourcePlugin, c.Source("segments"))
		}},
		{"segments empty list", `{"segments": []}`, func(t *testing.T, c *config.Config) {
			assert.Equal(t, []string{"from-file"}, c.Strings("segments"))
			assert.Equal(t, config.SourceFile, c.Source("segments"))
		}},
		{"separator", `{"separator": "|"}`, func(t *testing.T, c *config.Config) {
			assert.Equal(t, "|", c.String("separator"))
			assert.Equal(t, config.SourcePlugin, c.Source("separator"))
		}},
		{"separator empty", `{"separator": ""}`, func(t *testing.T, c *config.Config) {
			assert.Equal(t, "·", c.String("separator"))
			assert.Equal(t, config.SourceDefaults, c.Source("separator"))
		}},
		{"path_strip", `{"path_strip": ["~/code"]}`, func(t *testing.T, c *config.Config) {
			assert.Equal(t, []string{"~/code"}, c.Strings("path.strip"))
			assert.Equal(t, config.SourcePlugin, c.Source("path.strip"))
		}},
		{"path_worktrees", `{"path_worktrees": [".worktrees"]}`, func(t *testing.T, c *config.Config) {
			assert.Equal(t, []string{".worktrees"}, c.Strings("path.worktrees"))
			assert.Equal(t, config.SourcePlugin, c.Source("path.worktrees"))
		}},
		{"snapshot_path", `{"snapshot_path": "/tmp/u.json"}`, func(t *testing.T, c *config.Config) {
			assert.Equal(t, "/tmp/u.json", c.String("snapshot.path"))
			assert.Equal(t, config.SourcePlugin, c.Source("snapshot.path"))
		}},
		{"a list of the wrong type is ignored", `{"segments": "path"}`, func(t *testing.T, c *config.Config) {
			assert.NotEqual(t, []string{"path"}, c.Strings("segments"))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.check(t, resolve(t, settings(tt.options), "segments: [from-file]\n"))
		})
	}
}

func TestResolveToggles(t *testing.T) {
	t.Parallel()

	toggles := []struct{ option, key string }{
		{"path_branch", "path.branch"},
		{"path_emoji", "path.emoji"},
		{"model_effort", "model.effort"},
		{"snapshot_enabled", "snapshot.enabled"},
	}
	values := []struct {
		value      string
		fileValue  string
		want       bool
		wantSource string
	}{
		{"on", "false", true, config.SourcePlugin},
		{"off", "true", false, config.SourcePlugin},
		{"default", "false", false, config.SourceFile},
		{"default", "true", true, config.SourceFile},
		{"", "false", false, config.SourceFile},
		{"on", "", true, config.SourcePlugin},
		{"off", "", false, config.SourcePlugin},
		{"default", "", true, config.SourceDefaults},
	}
	for _, tg := range toggles {
		for _, v := range values {
			t.Run(fmt.Sprintf("%s %q file %q", tg.option, v.value, v.fileValue), func(t *testing.T) {
				t.Parallel()

				section, name, _ := strings.Cut(tg.key, ".")
				file := ""
				if v.fileValue != "" {
					file = fmt.Sprintf("%s:\n  %s: %s\n", section, name, v.fileValue)
				}

				cfg := resolve(t, settings(fmt.Sprintf(`{%q: %q}`, tg.option, v.value)), file)

				want := v.want
				if v.wantSource == config.SourceDefaults {
					want = tg.key != "path.emoji"
				}
				assert.Equal(t, want, cfg.Bool(tg.key))
				assert.Equal(t, v.wantSource, cfg.Source(tg.key))
			})
		}
	}
}

func TestResolveAnOptionKeepsTheSiblingKeysOfItsSection(t *testing.T) {
	t.Parallel()

	cfg := resolve(t, settings(`{"path_branch": "off"}`), "path:\n  emoji: true\n")

	assert.False(t, cfg.Bool("path.branch"))
	assert.True(t, cfg.Bool("path.emoji"))
	assert.Equal(t, config.SourceFile, cfg.Source("path.emoji"))
}

func TestResolveSettingsProblems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		settings   string
		wantIssues int
		wantActive bool
	}{
		{"missing file", "", 0, false},
		{"malformed JSON", `{"pluginConfigs": `, 1, false},
		{"wrong shape", `{"pluginConfigs": []}`, 1, false},
		{"no statusline entry", `{"enabledPlugins": {"other@m": true}, "pluginConfigs": {"other@m": {"options": {"theme": "saber"}}}}`, 0, false},
		{"disabled plugin is ignored", fmt.Sprintf(`{"enabledPlugins": {%q: false}, "pluginConfigs": {%q: {"options": {"theme": "saber"}}}}`, entry, entry), 0, false},
		{"not listed as enabled is ignored", fmt.Sprintf(`{"pluginConfigs": {%q: {"options": {"theme": "saber"}}}}`, entry), 0, false},
		{"options of another plugin sharing a prefix are ignored", `{"enabledPlugins": {"statusline-extra@m": true}, "pluginConfigs": {"statusline-extra@m": {"options": {"theme": "saber"}}}}`, 0, false},
		{"usable entry", settings(`{"theme": "saber"}`), 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := resolve(t, tt.settings, "theme: gradient\n")
			opts := cfg.PluginOptions()

			assert.Len(t, opts.Issues, tt.wantIssues)
			assert.Equal(t, tt.wantActive, opts.Active())
			if tt.wantActive {
				assert.Equal(t, "saber", cfg.String("theme"))
			} else {
				assert.Equal(t, "gradient", cfg.String("theme"))
				assert.Nil(t, cfg.Problem())
			}
		})
	}
}

func TestResolveSeveralEntriesUsesTheFirstBySortedKey(t *testing.T) {
	t.Parallel()

	text := `{"enabledPlugins": {"statusline@b": true, "statusline@a": true, "statusline@c": false},
	"pluginConfigs": {
		"statusline@b": {"options": {"theme": "bee"}},
		"statusline@a": {"options": {"theme": "ay"}},
		"statusline@c": {"options": {"theme": "sea"}}}}`

	cfg := resolve(t, text, "")
	opts := cfg.PluginOptions()

	assert.Equal(t, "ay", cfg.String("theme"))
	assert.Equal(t, "statusline@a", opts.Entry)
	require.Len(t, opts.Issues, 1)
	assert.Contains(t, opts.Issues[0], "statusline@a, statusline@b")
}

func TestResolveWithABrokenConfigFileStillAppliesOptions(t *testing.T) {
	t.Parallel()

	cfg := resolve(t, settings(`{"theme": "saber"}`), "theme: [unclosed\n")

	assert.Equal(t, "saber", cfg.String("theme"))
	assert.NotNil(t, cfg.Problem())
}

func TestResolveWithNoConfigDirectory(t *testing.T) {
	t.Parallel()

	cfg := config.Resolve(func(string) string { return "" })

	assert.Equal(t, "classic", cfg.String("theme"))
	assert.False(t, cfg.PluginOptions().Active())
	assert.Empty(t, cfg.PluginOptions().Issues)
}

func TestOptionsAreUnique(t *testing.T) {
	t.Parallel()

	names, keys := map[string]bool{}, map[string]bool{}
	for _, o := range config.Options() {
		assert.False(t, names[o.Name], o.Name)
		assert.False(t, keys[o.Key], o.Key)
		names[o.Name], keys[o.Key] = true, true
	}
}

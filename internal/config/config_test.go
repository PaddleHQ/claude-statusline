package config_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultsParse(t *testing.T) {
	t.Parallel()

	c := config.New("")
	assert.Equal(t, "path model ctx tokens cache", c.String("segments"))
	assert.Equal(t, "·", c.String("separator"))
	assert.Equal(t, "classic", c.String("theme"))
}

func TestDefaultKeys(t *testing.T) {
	t.Parallel()

	keys := config.DefaultKeys()
	slices.Sort(keys)
	want := []string{"segments", "separator", "theme"}
	assert.Equal(t, want, keys)
}

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want map[string]string
	}{
		{"trims keys and values", "  separator   =   |  ", map[string]string{"separator": "|"}},
		{"no spaces around equals", "segments=model", map[string]string{"segments": "model"}},
		{"ignores comments and blanks", "# a comment\n\n   # indented comment\ntheme = x", map[string]string{"theme": "x"}},
		{"last occurrence wins", "a.b = 1\na.b = 2", map[string]string{"a.b": "2"}},
		{"ignores lines without equals", "not a setting line", map[string]string{}},
		{"keeps later equals signs in the value", "separator = a=b", map[string]string{"separator": "a=b"}},
		{"handles CRLF line endings", "theme = x\r\nsegments = y\r\n", map[string]string{"theme": "x", "segments": "y"}},
		{"reads a final line without newline", "theme = x", map[string]string{"theme": "x"}},
		{"rejects invalid keys", "a.b.c = 1\nA.b = 1\na-b = 1\na..b = 1\n.b = 1\na. = 1\n1a = 1\n_a = 1\n = 1", map[string]string{}},
		{"accepts underscores and digits", "a1_b.c_2 = 1", map[string]string{"a1_b.c_2": "1"}},
		{"keeps shell metacharacters as data", "separator = $(touch x) `y` \"z\"\\", map[string]string{"separator": "$(touch x) `y` \"z\"\\"}},
		{"allows an empty value", "theme =", map[string]string{"theme": ""}},
		{"trailing comment is part of the value", "theme = x # y", map[string]string{"theme": "x # y"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, config.Parse(tt.text))
		})
	}
}

func TestString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		user string
		want string
	}{
		{"user value wins", "separator = |", "|"},
		{"empty user value falls back", "separator =", "·"},
		{"missing user value falls back", "", "·"},
		{"invalid key is ignored", "separator.x = |", "·"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, config.New(tt.user).String("separator"))
		})
	}
}

func TestParseNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in     string
		want   int
		wantOK bool
	}{
		{"0", 0, true},
		{"20", 20, true},
		{"08", 8, true},
		{"000000000000000020", 20, true},
		{"999999999999", 999999999999, true},
		{"1000000000000", 0, false},
		{"", 0, false},
		{"-3", 0, false},
		{"+3", 0, false},
		{"4x", 0, false},
		{"1e2", 0, false},
		{"5; touch x", 0, false},
		{"`touch x`", 0, false},
		{"1.5", 0, false},
		{" 3", 0, false},
	}
	for _, tt := range tests {
		got, ok := config.ParseNumber(tt.in)
		assert.Equal(t, tt.want, got, "ParseNumber(%q)", tt.in)
		assert.Equal(t, tt.wantOK, ok, "ParseNumber(%q)", tt.in)
	}
}

func TestIntAndBool(t *testing.T) {
	t.Parallel()

	defaults := map[string]string{"n": "7", "b": "true", "f": "false", "bad": "x"}
	c := config.NewWithDefaults(defaults, map[string]string{})
	tests := []struct {
		name string
		user map[string]string
		n    int
		b    bool
		f    bool
	}{
		{"defaults", map[string]string{}, 7, true, false},
		{"valid user values", map[string]string{"n": "08", "b": "false", "f": "true"}, 8, false, true},
		{"invalid user values", map[string]string{"n": "abc", "b": "maybe", "f": "TRUE"}, 7, true, false},
		{"negative number", map[string]string{"n": "-3"}, 7, true, false},
		{"too many digits", map[string]string{"n": "1234567890123"}, 7, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := config.NewWithDefaults(defaults, tt.user)
			assert.Equal(t, tt.n, c.Int("n"))
			assert.Equal(t, tt.b, c.Bool("b"))
			assert.Equal(t, tt.f, c.Bool("f"))
		})
	}

	assert.Zero(t, c.Int("bad"), "Int() with an invalid default")
	assert.Zero(t, c.Int("absent"), "Int() of an unknown key")
	assert.False(t, c.Bool("absent"), "Bool() of an unknown key")
}

func TestLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	require.NoError(t, os.WriteFile(path, []byte("separator = |\n"), 0o600))

	assert.Equal(t, "|", config.Load(path).String("separator"))
	assert.Equal(t, "·", config.Load(filepath.Join(dir, "missing")).String("separator"), "with no file")
	assert.Equal(t, "·", config.Load(dir).String("separator"), "with an unreadable file")
}

func TestPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"config dir wins", map[string]string{"CLAUDE_CONFIG_DIR": "/c", "HOME": "/h"}, "/c/claude-statusline/config"},
		{"falls back to HOME", map[string]string{"HOME": "/h"}, "/h/.claude/claude-statusline/config"},
		{"empty config dir falls back", map[string]string{"CLAUDE_CONFIG_DIR": "", "HOME": "/h"}, "/h/.claude/claude-statusline/config"},
		{"nothing set", map[string]string{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, config.Path(func(k string) string { return tt.env[k] }))
		})
	}
}

func TestPluginDefaults(t *testing.T) {
	t.Parallel()

	const plugin = "# a plugin key\nfoo.size = 3\nfoo.on = true\n"

	t.Run("layer under the user's config and over nothing else", func(t *testing.T) {
		t.Parallel()
		c := config.New("foo.size = 9", plugin)
		assert.Equal(t, 9, c.Int("foo.size"))
		assert.True(t, c.Bool("foo.on"))
		assert.Equal(t, "·", c.String("separator"), "global defaults stay")
	})

	t.Run("are listed with the global keys", func(t *testing.T) {
		t.Parallel()
		keys := config.DefaultKeys(plugin)
		assert.Contains(t, keys, "foo.size")
		assert.Contains(t, keys, "separator")
	})

	t.Run("are used by Load", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 3, config.Load(filepath.Join(t.TempDir(), "missing"), plugin).Int("foo.size"))
	})
}

func TestSection(t *testing.T) {
	t.Parallel()

	c := config.New("foo.size = 9\nfoo.on = false", "foo.size = 3\nfoo.on = true\nfoo.label = x\n")
	foo := c.Section("foo")
	assert.Equal(t, 9, foo.Int("size"))
	assert.False(t, foo.Bool("on"))
	assert.Equal(t, "x", foo.String("label"))
	assert.Equal(t, "·", c.Section("").String("separator"), "an empty name reads keys as they are")
	assert.Zero(t, c.Section("bar").Int("size"), "another section does not see foo's keys")
}

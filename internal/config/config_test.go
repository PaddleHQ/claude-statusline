package config_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultsParse(t *testing.T) {
	t.Parallel()

	c := config.New("")
	assert.Equal(t, []string{"model"}, c.Strings("segments"))
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

func TestUncommentedKeys(t *testing.T) {
	t.Parallel()

	const text = `# described
a: 1
b: 2
section:
  # described
  c: 3
  d: 4
other:
  # described
  e: [x]
`
	assert.Equal(t, []string{"b", "section.d"}, config.UncommentedKeys(text))
	assert.Empty(t, config.UncommentedKeys("a: [unclosed"))
	assert.Empty(t, config.UncommentedKeys(""))
	assert.Empty(t, config.UncommentedKeys("- a"))
}

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want map[string]any
	}{
		{"a mapping", "separator: '|'\nhooks:\n  ttl: 5", map[string]any{"separator": "|", "hooks": map[string]any{"ttl": 5}}},
		{"empty text", "", nil},
		{"comments only", "# nothing\n", nil},
		{"invalid YAML", "a: [", nil},
		{"duplicate key", "a: 1\na: 2", nil},
		{"a list is not a mapping", "- a\n- b", nil},
		{"a scalar is not a mapping", "just text", nil},
		{"non-string keys are dropped", "1: x\na: y", map[string]any{"a": "y"}},
		{"keeps shell metacharacters as data", `separator: "$(touch x) ` + "`y`" + ` \"z\"\\"`, map[string]any{"separator": "$(touch x) `y` \"z\"\\"}},
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
		{"user value wins", "separator: '|'", "|"},
		{"empty user value falls back", "separator: ''", "·"},
		{"null falls back", "separator:", "·"},
		{"missing user value falls back", "", "·"},
		{"a number falls back", "separator: 5", "·"},
		{"a bool falls back", "separator: true", "·"},
		{"a list falls back", "separator: [a]", "·"},
		{"a mapping falls back", "separator: {a: b}", "·"},
		{"a control character falls back", `separator: "a\tb"`, "·"},
		{"an escape falls back", `separator: "\e[31m"`, "·"},
		{"a dotted top-level key is unknown", "separator.x: '|'", "·"},
		{"a quoted string is text", `separator: "'; rm -rf /"`, "'; rm -rf /"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, config.New(tt.user).String("separator"))
		})
	}
}

func TestStrings(t *testing.T) {
	t.Parallel()

	const def = "segments: [a, b]\n"
	tests := []struct {
		name string
		user string
		want []string
	}{
		{"user list wins", "segments: [x, 'y z']", []string{"x", "y z"}},
		{"block sequence", "segments:\n  - x\n  - y", []string{"x", "y"}},
		{"empty entries are dropped", "segments: [x, '', y]", []string{"x", "y"}},
		{"an empty list falls back", "segments: []", []string{"a", "b"}},
		{"only empty entries falls back", "segments: ['']", []string{"a", "b"}},
		{"a scalar falls back", "segments: x", []string{"a", "b"}},
		{"a non-string entry falls back", "segments: [x, 5]", []string{"a", "b"}},
		{"a null entry falls back", "segments: [x, ~]", []string{"a", "b"}},
		{"a control character falls back", `segments: ["x\ny"]`, []string{"a", "b"}},
		{"missing falls back", "", []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, config.New(tt.user, def).Strings("segments"))
		})
	}

	assert.Nil(t, config.New("", def).Strings("absent"), "Strings() of an unknown key")
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

	defaults := map[string]any{"n": 7, "b": true, "f": false, "bad": "x"}
	c := config.NewWithDefaults(defaults, nil)
	tests := []struct {
		name string
		user map[string]any
		n    int
		b    bool
		f    bool
	}{
		{"defaults", nil, 7, true, false},
		{"valid user values", map[string]any{"n": 8, "b": false, "f": true}, 8, false, true},
		{"zero is valid", map[string]any{"n": 0}, 0, true, false},
		{"int64 and uint64", map[string]any{"n": int64(9)}, 9, true, false},
		{"the largest uint64 is too big", map[string]any{"n": uint64(1) << 63}, 7, true, false},
		{"a uint64 in range", map[string]any{"n": uint64(11)}, 11, true, false},
		{"wrong types", map[string]any{"n": "abc", "b": "maybe", "f": "true"}, 7, true, false},
		{"a float", map[string]any{"n": 2.5}, 7, true, false},
		{"negative number", map[string]any{"n": -3}, 7, true, false},
		{"negative int64", map[string]any{"n": int64(-3)}, 7, true, false},
		{"too many digits", map[string]any{"n": 1234567890123}, 7, true, false},
		{"the largest accepted number", map[string]any{"n": 999999999999}, 999999999999, true, false},
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

func TestYAMLTypes(t *testing.T) {
	t.Parallel()

	const def = "sec:\n  n: 7\n  b: true\n  s: x\n  l: [a]\n"
	tests := []struct {
		name string
		user string
		n    int
		b    bool
		s    string
		l    []string
	}{
		{"all set", "sec: {n: 3, b: false, s: y, l: [z]}", 3, false, "y", []string{"z"}},
		{"one key keeps the other defaults", "sec: {n: 3}", 3, true, "x", []string{"a"}},
		{"the section is not a mapping", "sec: 3", 7, true, "x", []string{"a"}},
		{"bool words are strings, not bools", "sec: {b: yes}", 7, true, "x", []string{"a"}},
		{"a quoted number is a string", "sec: {n: '3'}", 7, true, "x", []string{"a"}},
		{"a number is not a string", "sec: {s: 3}", 7, true, "x", []string{"a"}},
		{"a hex number is a number", "sec: {n: 0x10}", 16, true, "x", []string{"a"}},
		{"an invalid file applies nothing", "sec: {n: 3\n", 7, true, "x", []string{"a"}},
		{"a flat dotted key is unknown", "sec.n: 3", 7, true, "x", []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := config.New(tt.user, def).Section("sec")
			assert.Equal(t, tt.n, c.Int("n"))
			assert.Equal(t, tt.b, c.Bool("b"))
			assert.Equal(t, tt.s, c.String("s"))
			assert.Equal(t, tt.l, c.Strings("l"))
		})
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("separator: '|'\n"), 0o600))
	big := filepath.Join(dir, "big.yaml")
	require.NoError(t, os.WriteFile(big, []byte("separator: '|'\n#"+strings.Repeat("x", 1<<20)), 0o600))

	assert.Equal(t, "|", config.Load(path).String("separator"))
	assert.Equal(t, "·", config.Load(filepath.Join(dir, "missing")).String("separator"), "with no file")
	assert.Equal(t, "·", config.Load(dir).String("separator"), "with an unreadable file")
	assert.Equal(t, "·", config.Load(big).String("separator"), "with an oversized file")
}

func TestPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"config dir wins", map[string]string{"CLAUDE_CONFIG_DIR": "/c", "HOME": "/h"}, "/c/claude-statusline/config.yaml"},
		{"falls back to HOME", map[string]string{"HOME": "/h"}, "/h/.claude/claude-statusline/config.yaml"},
		{"empty config dir falls back", map[string]string{"CLAUDE_CONFIG_DIR": "", "HOME": "/h"}, "/h/.claude/claude-statusline/config.yaml"},
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

	const plugin = "foo:\n  # a plugin key\n  size: 3\n  on: true\n"

	t.Run("layer under the user's config and over nothing else", func(t *testing.T) {
		t.Parallel()
		c := config.New("foo: {size: 9}", plugin)
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

	c := config.New("foo: {size: 9, on: false}", "foo:\n  size: 3\n  on: true\n  label: x\n  list: [a]\n")
	foo := c.Section("foo")
	assert.Equal(t, 9, foo.Int("size"))
	assert.False(t, foo.Bool("on"))
	assert.Equal(t, "x", foo.String("label"))
	assert.Equal(t, []string{"a"}, foo.Strings("list"))
	assert.Equal(t, "·", c.Section("").String("separator"), "an empty name reads keys as they are")
	assert.Zero(t, c.Section("bar").Int("size"), "another section does not see foo's keys")
}

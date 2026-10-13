package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDefaultsParse(t *testing.T) {
	t.Parallel()

	c := New("")
	if got := c.String("segments"); got != "model" {
		t.Errorf("segments = %q, want model", got)
	}
	if got := c.String("separator"); got != "·" {
		t.Errorf("separator = %q, want ·", got)
	}
	if got := c.String("theme"); got != "default" {
		t.Errorf("theme = %q, want default", got)
	}
	if !c.Bool("model.effort") {
		t.Error("model.effort = false, want true")
	}
}

func TestDefaultKeys(t *testing.T) {
	t.Parallel()

	keys := DefaultKeys()
	slices.Sort(keys)
	want := []string{"model.effort", "segments", "separator", "theme"}
	if !slices.Equal(keys, want) {
		t.Errorf("DefaultKeys() = %v, want %v", keys, want)
	}
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
			got := parse(tt.text)
			if len(got) != len(tt.want) {
				t.Fatalf("parse() = %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parse()[%q] = %q, want %q", k, got[k], v)
				}
			}
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
			if got := New(tt.user).String("separator"); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
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
		{"1.5", 0, false},
		{" 3", 0, false},
	}
	for _, tt := range tests {
		got, ok := ParseNumber(tt.in)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("ParseNumber(%q) = %d, %v; want %d, %v", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestIntAndBool(t *testing.T) {
	t.Parallel()

	c := &Config{
		defaults: map[string]string{"n": "7", "b": "true", "f": "false", "bad": "x"},
		user:     map[string]string{},
	}
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
			c := &Config{defaults: c.defaults, user: tt.user}
			if got := c.Int("n"); got != tt.n {
				t.Errorf("Int() = %d, want %d", got, tt.n)
			}
			if got := c.Bool("b"); got != tt.b {
				t.Errorf("Bool(b) = %v, want %v", got, tt.b)
			}
			if got := c.Bool("f"); got != tt.f {
				t.Errorf("Bool(f) = %v, want %v", got, tt.f)
			}
		})
	}

	if got := c.Int("bad"); got != 0 {
		t.Errorf("Int() with an invalid default = %d, want 0", got)
	}
	if got := c.Int("absent"); got != 0 {
		t.Errorf("Int() of an unknown key = %d, want 0", got)
	}
	if c.Bool("absent") {
		t.Error("Bool() of an unknown key = true, want false")
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte("separator = |\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := Load(path).String("separator"); got != "|" {
		t.Errorf("separator = %q, want |", got)
	}
	if got := Load(filepath.Join(dir, "missing")).String("separator"); got != "·" {
		t.Errorf("separator with no file = %q, want ·", got)
	}
	if got := Load(dir).String("separator"); got != "·" {
		t.Errorf("separator with an unreadable file = %q, want ·", got)
	}
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
			if got := Path(func(k string) string { return tt.env[k] }); got != tt.want {
				t.Errorf("Path() = %q, want %q", got, tt.want)
			}
		})
	}
}

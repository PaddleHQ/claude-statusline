package config_test

import (
	"strings"
	"testing"
	"unicode"

	"github.com/PaddleHQ/claude-statusline/internal/config"

	"github.com/stretchr/testify/assert"
)

func FuzzLoad(f *testing.F) {
	f.Add("separator: '|'")
	f.Add("# a comment\n\n   # indented comment\ntheme: x")
	f.Add("a:\n  b: 1\n  b: 2")
	f.Add("separator: \"$(touch x) `y` \\\"z\\\"\\\\\"")
	f.Add("segments: [model, ctx]\nctx:\n  width: 10\nnot a setting line")
	f.Add("segments: [model, 5]\nhooks: {ttl: -1, timeout: 99999999999999}")
	f.Add("snapshot: {enabled: maybe, path: [a]}\nseparator: \"a\\nb\"")
	f.Add("a: &x [*x]")
	f.Add("- list\n- not a mapping")
	f.Add("separator: [unclosed")
	f.Add("1: x\n? [a]\n: b")

	defaults := config.New("")
	f.Fuzz(func(t *testing.T, text string) {
		cfg := config.New(text)
		for _, key := range config.DefaultKeys() {
			assert.NotPanics(t, func() {
				s, l, n, b := cfg.String(key), cfg.Strings(key), cfg.Int(key), cfg.Bool(key)
				assert.False(t, strings.ContainsFunc(s, unicode.IsControl), "String(%q) holds a control character", key)
				for _, item := range l {
					assert.NotEmpty(t, item, "Strings(%q)", key)
					assert.False(t, strings.ContainsFunc(item, unicode.IsControl), "Strings(%q) holds a control character", key)
				}
				assert.GreaterOrEqual(t, n, 0, "Int(%q)", key)
				assert.LessOrEqual(t, n, 999_999_999_999, "Int(%q)", key)
				_ = b
			})
		}
		if config.Parse(text) == nil {
			for _, key := range config.DefaultKeys() {
				assert.Equal(t, defaults.String(key), cfg.String(key), "String(%q) without a usable file", key)
				assert.Equal(t, defaults.Strings(key), cfg.Strings(key), "Strings(%q) without a usable file", key)
				assert.Equal(t, defaults.Int(key), cfg.Int(key), "Int(%q) without a usable file", key)
				assert.Equal(t, defaults.Bool(key), cfg.Bool(key), "Bool(%q) without a usable file", key)
			}
		}
	})
}

func FuzzParseNumber(f *testing.F) {
	for _, s := range []string{"", "0", "42", "000000000000042", "-1", "1e3", "999999999999", "9999999999999", "１２"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		n, ok := config.ParseNumber(s)
		if ok {
			assert.GreaterOrEqual(t, n, 0)
		}
	})
}

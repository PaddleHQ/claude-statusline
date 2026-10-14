package config_test

import (
	"regexp"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"

	"github.com/stretchr/testify/assert"
)

var documentedKey = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)?$`)

func FuzzParse(f *testing.F) {
	f.Add("  separator   =   |  ")
	f.Add("# a comment\n\n   # indented comment\ntheme = x")
	f.Add("a.b = 1\na.b = 2")
	f.Add("separator = a=b")
	f.Add("theme = x\r\nsegments = y\r\n")
	f.Add("a.b.c = 1\nA.b = 1\na-b = 1\na..b = 1\n.b = 1\na. = 1\n1a = 1\n_a = 1\n = 1")
	f.Add("separator = $(touch x) `y` \"z\"\\")
	f.Add("segments = model, ctx\nbar.width = 10\nnot a setting line")

	f.Fuzz(func(t *testing.T, text string) {
		for key, value := range config.Parse(text) {
			assert.Regexp(t, documentedKey, key)
			assert.NotContains(t, value, "\n", "value for %q", key)
		}
		cfg := config.New(text)
		for _, key := range config.DefaultKeys() {
			_ = cfg.String(key)
			_ = cfg.Int(key)
			_ = cfg.Bool(key)
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

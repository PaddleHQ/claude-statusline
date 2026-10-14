package theme_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/PaddleHQ/claude-statusline/internal/theme"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var onlySGR = regexp.MustCompile("^(\x1b\\[[0-9;]*m)*$")

func FuzzParseColor(f *testing.F) {
	for _, s := range []string{"", "bold 33", "dim", "green", "bright-red", "#ff8800", "bold italic underline cyan", "300", "#12345", "red blue", "\x1b[2J", "bold\x1b[31m"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, spec string) {
		seq, err := theme.ParseColor(spec)
		if err != nil {
			assert.Empty(t, seq)
			return
		}
		assert.Regexp(t, onlySGR, seq)
	})
}

func FuzzOpen(f *testing.F) {
	entries, err := os.ReadDir("../../themes")
	require.NoError(f, err)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".yaml") {
			data, err := os.ReadFile("../../themes/" + e.Name())
			require.NoError(f, err)
			f.Add(data)
		}
	}
	f.Add([]byte(""))
	f.Add([]byte("colors: {}\nbar: {}\n"))
	f.Add([]byte("colors: &a {text: \"\"}\nbar: *a\n"))

	path := filepath.Join(f.TempDir(), "theme.yaml")

	f.Fuzz(func(t *testing.T, data []byte) {
		require.NoError(t, os.WriteFile(path, data, 0o600))
		th, err := theme.Open(path, "", "")
		if err != nil {
			return
		}
		for name, seq := range map[string]string{
			"text": th.TextColor, "muted": th.Muted, "good": th.Good, "warn": th.WarnColor, "bad": th.BadColor,
			"marker": th.Marker, "path": th.PathColor, "reset": th.Reset, "empty_color": th.EmptyColor, "marker_color": th.MarkerColor,
		} {
			assert.Regexp(t, onlySGR, seq, name)
		}
		glyphs := []string{th.Full, th.Empty, th.MarkerGlyph}
		for _, p := range th.Partials {
			if p != "" {
				glyphs = append(glyphs, p)
			}
		}
		for _, g := range glyphs {
			assert.NotEmpty(t, g)
			assert.False(t, strings.ContainsFunc(g, unicode.IsControl), "glyph %q holds a control character", g)
		}
	})
}

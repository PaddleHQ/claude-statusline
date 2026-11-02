package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fixturesDir = "../../features"
	imagesDir   = "../../docs/images"
	regenerate  = "run `go run ./tools/screenshots` from the repository root and commit the result"
)

func TestScreenshotsUpToDate(t *testing.T) {
	t.Parallel()

	images, err := Generate(fixturesDir)
	require.NoError(t, err)
	require.NotEmpty(t, images, "no screenshot fixtures found")

	want := make(map[string]bool, len(images))
	for _, img := range images {
		want[img.Name] = true
		got, err := os.ReadFile(filepath.Join(imagesDir, img.Name))
		if !assert.NoError(t, err, "docs/images/%s is missing: %s", img.Name, regenerate) {
			continue
		}
		assert.Equal(t, string(img.SVG), string(got), "docs/images/%s is out of date: %s", img.Name, regenerate)
	}

	committed, err := filepath.Glob(filepath.Join(imagesDir, "*.svg"))
	require.NoError(t, err)
	for _, path := range committed {
		assert.True(t, want[filepath.Base(path)], "docs/images/%s has no fixture: delete it or add the fixture", filepath.Base(path))
	}
}

func TestGenerateRendersTheRealStatusLine(t *testing.T) {
	t.Parallel()

	images, err := Generate(fixturesDir)
	require.NoError(t, err)

	byName := make(map[string]string, len(images))
	for _, img := range images {
		byName[img.Name] = string(img.SVG)
	}
	assert.Contains(t, byName["hero-dark.svg"], "feature/retry-backoff")
	assert.Contains(t, byName["custom-segment-dark.svg"], "backup stale")
	assert.Contains(t, byName["subagent-dark.svg"], "reviewer")
	assert.Contains(t, byName["narrow-dark.svg"], "↳")
	assert.NotRegexp(t, `<tspan[^>]*fill=`, byName["gradient-no-color-dark.svg"], "NO_COLOR must remove every color from the text")
	assert.NotRegexp(t, `<tspan[^>]*fill=`, byName["saber-no-color-dark.svg"], "NO_COLOR must remove every color from the text")
}

func TestGenerateIsRepeatable(t *testing.T) {
	t.Parallel()

	first, err := Generate(fixturesDir)
	require.NoError(t, err)
	second, err := Generate(fixturesDir)
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestWriteSavesEveryImage(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "images")
	images := []Image{{Name: "a-dark.svg", SVG: []byte("<svg/>")}}
	require.NoError(t, Write(dir, images))

	got, err := os.ReadFile(filepath.Join(dir, "a-dark.svg"))
	require.NoError(t, err)
	assert.Equal(t, "<svg/>", string(got))
}

func TestSVGEmbedsTheSymbolsFontOnlyForIcons(t *testing.T) {
	t.Parallel()

	const fontFace = `@font-face{font-family:'Symbols Nerd Font Mono';src:url(data:font/woff2;base64,`
	tests := []struct {
		name  string
		in    string
		embed bool
	}{
		{"an icon in the Basic Multilingual Plane", " a", true},
		{"an icon in a supplementary Private Use Area", "\U000F06A9", true},
		{"no icons", "ctx: ▰▱ 10%", false},
		{"the first code point of the Private Use Area", "\uE000", true},
		{"the last code point of the Private Use Area", "\uF8FF", true},
		{"the first code point of the supplementary Private Use Area", "\U000F0000", true},
		{"the code point before the surrogates", "\uD7FF", false},
		{"the code point after the Private Use Area", "\uF900", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svg := string(Parse(tt.in).SVG(Dark))
			if tt.embed {
				assert.Contains(t, svg, fontFace)
				assert.Regexp(t, `<rect [^>]*/>\n<defs><style>@font-face`, svg, "the font is defined right after the background")
				assert.Regexp(t, `font-family="[^"]*'Symbols Nerd Font Mono', monospace"`, svg)
				return
			}
			assert.NotContains(t, svg, "@font-face")
			assert.NotContains(t, svg, "Symbols Nerd Font Mono")
		})
	}
}

func TestSVGKeepsTextAfterAnIconOnTheGrid(t *testing.T) {
	t.Parallel()

	svg := string(Parse("( main)").SVG(Dark))
	assert.Contains(t, svg, `<tspan x="16">(</tspan><tspan x="25">`+""+`</tspan><tspan x="34"> main)</tspan>`,
		"an icon is wider than a cell, so the text after it starts its own tspan at its column")
}

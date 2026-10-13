package screenshot_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/screenshot"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fixturesDir = "../../testdata/screenshots"
	imagesDir   = "../../docs/images"
	regenerate  = "run `go run ./tools/screenshots` from the repository root and commit the result"
)

func TestScreenshotsUpToDate(t *testing.T) {
	t.Parallel()

	images, err := screenshot.Generate(fixturesDir)
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

	images, err := screenshot.Generate(fixturesDir)
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
}

func TestGenerateIsRepeatable(t *testing.T) {
	t.Parallel()

	first, err := screenshot.Generate(fixturesDir)
	require.NoError(t, err)
	second, err := screenshot.Generate(fixturesDir)
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestWriteSavesEveryImage(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "images")
	images := []screenshot.Image{{Name: "a-dark.svg", SVG: []byte("<svg/>")}}
	require.NoError(t, screenshot.Write(dir, images))

	got, err := os.ReadFile(filepath.Join(dir, "a-dark.svg"))
	require.NoError(t, err)
	assert.Equal(t, "<svg/>", string(got))
}

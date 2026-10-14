package rendertest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/rendertest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sample = "Shows a model name\n-- input.json --\n{}\n-- env --\n# comment\nCOLUMNS=50\n-- output --\nold\n-- expected_cache --\n{\"a\":1}\n"

func load(t *testing.T, content string) (c *rendertest.Case, path string) {
	t.Helper()

	dir := t.TempDir()
	path = filepath.Join(dir, "case.txtar")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	cases, err := rendertest.Load(dir)
	require.NoError(t, err)
	require.Len(t, cases, 1)
	return cases[0], path
}

func TestLoadReadsSections(t *testing.T) {
	t.Parallel()

	c, _ := load(t, sample)
	assert.Equal(t, "case", c.Name)
	assert.Equal(t, "Shows a model name", c.Description())

	want, ok := c.Expected(rendertest.Output)
	assert.True(t, ok)
	assert.Equal(t, "old", want, "exactly one trailing newline is trimmed")

	_, ok = c.Section(rendertest.Config)
	assert.False(t, ok)
	_, err := c.Required(rendertest.Config)
	require.ErrorIs(t, err, rendertest.ErrMissingSection)

	env := c.Env(map[string]string{"COLUMNS": "200", "HOME": "/h"})
	assert.Equal(t, map[string]string{"COLUMNS": "50", "HOME": "/h"}, env)
}

func TestExpectedKeepsOtherTrailingNewlines(t *testing.T) {
	t.Parallel()

	c, _ := load(t, "-- input.json --\n{}\n-- output --\nrow\n\n")
	want, _ := c.Expected(rendertest.Output)
	assert.Equal(t, "row\n", want)
}

func TestUpdateRewritesOnlyChangedSections(t *testing.T) {
	t.Parallel()

	c, path := load(t, sample)
	require.NoError(t, c.Update(map[string]string{
		rendertest.Output:        "new",
		rendertest.ExpectedCache: " {\"a\":1}\n",
		rendertest.Config:        "ignored",
	}))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Shows a model name\n-- input.json --\n{}\n-- env --\n# comment\nCOLUMNS=50\n-- output --\nnew\n-- expected_cache --\n{\"a\":1}\n", string(got))
}

func TestUpdateLeavesAMatchingFileUntouched(t *testing.T) {
	t.Parallel()

	c, path := load(t, "-- input.json --\n{}\n-- output --\nsame")
	require.NoError(t, os.Remove(path))
	require.NoError(t, c.Update(map[string]string{rendertest.Output: "same"}))
	assert.NoFileExists(t, path)
}

func TestUpdateWritesEmptyOutputWithoutABlankLine(t *testing.T) {
	t.Parallel()

	c, path := load(t, "-- input.json --\n{}\n-- output --\nold\n")
	require.NoError(t, c.Update(map[string]string{rendertest.Output: ""}))
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "-- input.json --\n{}\n-- output --\n", string(got))
}

func TestWriteTreeCreatesFilesAndDirectories(t *testing.T) {
	t.Parallel()

	c, _ := load(t, "-- input.json --\n{}\n-- tree --\ndir/\n+x bin/run.sh=#!/bin/sh\\nexit 0\nnote.txt=at @TMP@\n-- output --\n")
	tmp := t.TempDir()
	require.NoError(t, c.WriteTree(tmp))

	assert.DirExists(t, filepath.Join(tmp, "dir"))
	script, err := os.ReadFile(filepath.Join(tmp, "bin", "run.sh"))
	require.NoError(t, err)
	assert.Equal(t, "#!/bin/sh\nexit 0\n", string(script))
	info, err := os.Stat(filepath.Join(tmp, "bin", "run.sh"))
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&0o100, "+x makes the file executable")
	note, err := os.ReadFile(filepath.Join(tmp, "note.txt"))
	require.NoError(t, err)
	assert.Equal(t, "at "+tmp+"\n", string(note))
}

func TestLoadFailsWithoutCases(t *testing.T) {
	t.Parallel()

	_, err := rendertest.Load(t.TempDir())
	require.ErrorIs(t, err, rendertest.ErrNoCases)
}

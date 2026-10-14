package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/rendertest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "rewrite the output and expected_cache sections of the render tests from the current code")

// Each testdata/render/<case>.txtar holds an input.json and an output section, plus optional config, env, cache
// (seeded before the run), tree (files created in the temp directory before the run) and expected_cache
// (compared after it). Escape bytes in the output are compared as the text \e, and @TMP@ stands for the case's
// temp directory in input.json, config, tree and output. See CONTRIBUTING.md for the format.
func TestRender(t *testing.T) {
	t.Parallel()

	cases, err := rendertest.Load(filepath.Join("testdata", "render"))
	require.NoError(t, err)

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()
			runRender(t, c)
		})
	}
}

func runRender(t *testing.T, c *rendertest.Case) {
	t.Helper()

	tmp := t.TempDir()
	env := c.Env(map[string]string{
		"HOME":                    tmp,
		"CLAUDE_CONFIG_DIR":       tmp,
		"COLUMNS":                 "200",
		"STATUSLINE_NOW":          "1760000000",
		"STATUSLINE_HOOK_TIMEOUT": "30",
	})
	require.NoError(t, c.WriteConfig(tmp))
	cacheFile, err := c.WriteCache(tmp)
	require.NoError(t, err)
	require.NoError(t, c.WriteTree(tmp))

	input, err := c.Input(tmp)
	require.NoError(t, err)
	_, err = c.Required(rendertest.Output)
	require.NoError(t, err)

	var out bytes.Buffer
	run(bytes.NewReader(input), &out, func(k string) string { return env[k] })

	got := strings.ReplaceAll(out.String(), "\033", `\e`)
	gotCache, _ := os.ReadFile(cacheFile)
	if *update {
		require.NoError(t, c.Update(map[string]string{
			rendertest.Output:        strings.ReplaceAll(got, tmp, rendertest.TmpMarker),
			rendertest.ExpectedCache: string(gotCache),
		}))
	} else {
		want, _ := c.Expected(rendertest.Output)
		assert.Equal(t, strings.ReplaceAll(want, rendertest.TmpMarker, tmp), got)
		if wantCache, ok := c.Section(rendertest.ExpectedCache); ok {
			assert.Equal(t, strings.TrimSpace(wantCache), strings.TrimSpace(string(gotCache)), "cache mismatch")
		}
	}
	assert.NotRegexp(t, "[\x00-\x09\x0b-\x1a\x1c-\x1f\x7f]", out.String(), "output contains a control character other than newline and escape")
	assert.NoFileExists(t, filepath.Join(tmp, "pwned"), "a config value was executed")
}

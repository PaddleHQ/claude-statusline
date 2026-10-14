package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/rendertest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Each testdata/help/<case>.txtar holds an output section, plus optional config, env and tree: the text
// `claude-statusline help` prints for that setup, with @TMP@ standing for the case's temp directory.
func TestHelp(t *testing.T) {
	t.Parallel()

	cases, err := rendertest.Load(filepath.Join("testdata", "help"))
	require.NoError(t, err)

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()
			runHelp(t, c)
		})
	}
}

func runHelp(t *testing.T, c *rendertest.Case) {
	t.Helper()

	tmp := t.TempDir()
	env := c.Env(map[string]string{"HOME": tmp, "CLAUDE_CONFIG_DIR": tmp})
	require.NoError(t, c.WriteConfig(tmp))
	require.NoError(t, c.WriteTree(tmp))
	_, err := c.Required(rendertest.Output)
	require.NoError(t, err)

	var out bytes.Buffer
	execute([]string{"help"}, strings.NewReader(""), &out, func(k string) string { return env[k] })

	got := strings.ReplaceAll(out.String(), tmp, rendertest.TmpMarker)
	if *update {
		require.NoError(t, c.Update(map[string]string{rendertest.Output: strings.TrimSuffix(got, "\n")}))
	} else {
		want, _ := c.Expected(rendertest.Output)
		assert.Equal(t, want, strings.TrimSuffix(got, "\n"))
	}
	assert.NotContains(t, out.String(), "\033", "help is plain text")
}

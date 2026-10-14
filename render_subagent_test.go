package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/rendertest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Each testdata/subagent/<case>.txtar holds an input.json (what Claude Code sends the subagent command) and an
// output section, plus optional config and env. Each output row is shown as its raw id, a tab and its content,
// with escape bytes written as \e. Rows were checked against an independent implementation of the same contract.
func TestRenderSubagent(t *testing.T) {
	t.Parallel()

	cases, err := rendertest.Load(filepath.Join("testdata", "subagent"))
	require.NoError(t, err)

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()
			runRenderSubagent(t, c)
		})
	}
}

func runRenderSubagent(t *testing.T, c *rendertest.Case) {
	t.Helper()

	tmp := t.TempDir()
	env := c.Env(map[string]string{"HOME": tmp, "CLAUDE_CONFIG_DIR": tmp})
	require.NoError(t, c.WriteConfig(tmp))

	input, err := c.Input(tmp)
	require.NoError(t, err)
	_, err = c.Required(rendertest.Output)
	require.NoError(t, err)

	var out bytes.Buffer
	execute([]string{"subagent"}, bytes.NewReader(input), &out, func(k string) string { return env[k] })

	var rows, contents []string
	for line := range strings.SplitSeq(strings.TrimSuffix(out.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		var row struct {
			ID      json.RawMessage `json:"id"`
			Content string          `json:"content"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &row), "output line %q", line)
		contents = append(contents, row.Content)
		rows = append(rows, string(row.ID)+"\t"+strings.ReplaceAll(row.Content, "\033", `\e`))
	}
	got := strings.Join(rows, "\n")
	if *update {
		require.NoError(t, c.Update(map[string]string{rendertest.Output: got}))
	} else {
		want, _ := c.Expected(rendertest.Output)
		assert.Equal(t, want, got)
	}
	assert.NotRegexp(t, "[\x00-\x09\x0b-\x1a\x1c-\x1f\x7f]", strings.Join(contents, ""), "content contains a control character other than escape")
	assert.NoFileExists(t, filepath.Join(tmp, "usage-latest.json"), "the subagent command must not write the usage snapshot")
}

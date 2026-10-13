package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Each testdata/golden-subagent/<case>/ holds input.json (what Claude Code sends the subagent command) and
// expected, plus optional config and env. Each output row is shown as its raw id, a tab and its content, with
// escape bytes written as \e. Rows were checked against an independent implementation of the same contract.
func TestGoldenSubagent(t *testing.T) {
	t.Parallel()

	cases, err := filepath.Glob(filepath.Join("testdata", "golden-subagent", "*"))
	require.NoError(t, err)
	require.NotEmpty(t, cases, "no golden cases found")

	for _, dir := range cases {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			t.Parallel()
			runGoldenSubagent(t, dir)
		})
	}
}

func runGoldenSubagent(t *testing.T, dir string) {
	t.Helper()

	tmp := t.TempDir()
	env := map[string]string{"HOME": tmp, "CLAUDE_CONFIG_DIR": tmp}
	if data, err := os.ReadFile(filepath.Join(dir, "env")); err == nil {
		for line := range strings.SplitSeq(string(data), "\n") {
			if k, v, ok := strings.Cut(line, "="); ok && !strings.HasPrefix(k, "#") {
				env[k] = v
			}
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, "config")); err == nil {
		appDir := filepath.Join(tmp, "claude-statusline")
		require.NoError(t, os.MkdirAll(appDir, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(appDir, "config"), data, 0o600))
	}

	input, err := os.ReadFile(filepath.Join(dir, "input.json"))
	require.NoError(t, err)
	want, err := os.ReadFile(filepath.Join(dir, "expected"))
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
	assert.Equal(t, strings.TrimSuffix(string(want), "\n"), strings.Join(rows, "\n"))
	assert.NotRegexp(t, "[\x00-\x09\x0b-\x1a\x1c-\x1f\x7f]", strings.Join(contents, ""), "content contains a control character other than escape")
	assert.NoFileExists(t, filepath.Join(tmp, "usage-latest.json"), "the subagent command must not write the usage snapshot")
}

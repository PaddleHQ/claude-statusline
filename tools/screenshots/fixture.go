package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/rendertest"
	"github.com/PaddleHQ/claude-statusline/internal/statusline"
	"github.com/PaddleHQ/claude-statusline/internal/subagent"
)

const (
	fixedNow       = "1760000000"
	hookTimeout    = "30"
	defaultColumns = "200"
	subagentName   = "subagent"
)

// Fixture is a txtar file under testdata/screenshots. It follows the conventions of testdata/render: an
// input.json, and optionally config, env and tree sections. A command section holding "subagent" renders the
// agent panel rows instead of the status line.
type Fixture struct {
	c *rendertest.Case
}

// Open returns the fixture for a parsed case.
func Open(c *rendertest.Case) *Fixture {
	return &Fixture{c: c}
}

// Name is the fixture's file name without its extension, which names its images.
func (f *Fixture) Name() string {
	return f.c.Name
}

// Render runs the real status line against the fixture with a private home, a fixed clock and a fixed width,
// and returns what it prints as terminal text, one line per row.
func (f *Fixture) Render() (string, error) {
	tmp, err := os.MkdirTemp("", "screenshot-")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	env := f.c.Env(map[string]string{
		"HOME":              tmp,
		"CLAUDE_CONFIG_DIR": tmp,
		"COLUMNS":           defaultColumns,
		"STATUSLINE_NOW":    fixedNow,

		"STATUSLINE_HOOK_TIMEOUT": hookTimeout,
	})
	if err := f.c.WriteConfig(tmp); err != nil {
		return "", err
	}
	if err := f.c.WriteTree(tmp); err != nil {
		return "", err
	}

	input, err := f.c.Input(tmp)
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	getenv := func(k string) string { return env[k] }

	if f.isSubagent() {
		return subagentRows(subagent.Build(input, getenv))
	}
	return statusline.Build(input, getenv), nil
}

func (f *Fixture) isSubagent() bool {
	data, _ := f.c.Section(rendertest.Command)
	return strings.TrimSpace(data) == subagentName
}

// subagentRows turns the JSON lines the subagent command prints into the rows Claude Code would draw.
func subagentRows(out string) (string, error) {
	var rows []string
	for line := range strings.SplitSeq(strings.TrimSuffix(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		var row struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(bytes.NewReader([]byte(line))).Decode(&row); err != nil {
			return "", fmt.Errorf("decode subagent row %q: %w", line, err)
		}
		rows = append(rows, row.Content)
	}
	return strings.Join(rows, "\n"), nil
}

package screenshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/statusline"
	"github.com/PaddleHQ/claude-statusline/internal/subagent"
)

const (
	fixedNow       = "1760000000"
	hookTimeout    = "30"
	defaultColumns = "200"
	subagentName   = "subagent"
	privateDir     = 0o700
	privateFile    = 0o600
	executableFile = 0o700
	tmpMarker      = "@TMP@"
)

// Fixture is a directory under testdata/screenshots. It follows the conventions of testdata/golden: input.json,
// and optionally config, env and tree. A file named command holding "subagent" renders the agent panel rows
// instead of the status line.
type Fixture struct {
	dir string
}

// Open returns the fixture in dir.
func Open(dir string) *Fixture {
	return &Fixture{dir: dir}
}

// Name is the fixture's directory name, which names its images.
func (f *Fixture) Name() string {
	return filepath.Base(f.dir)
}

// Render runs the real status line against the fixture with a private home, a fixed clock and a fixed width,
// and returns what it prints as terminal text, one line per row.
func (f *Fixture) Render() (string, error) {
	tmp, err := os.MkdirTemp("", "screenshot-")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	env := map[string]string{
		"HOME":              tmp,
		"CLAUDE_CONFIG_DIR": tmp,
		"COLUMNS":           defaultColumns,
		"STATUSLINE_NOW":    fixedNow,

		"STATUSLINE_HOOK_TIMEOUT": hookTimeout,
	}
	f.readEnv(env)
	if err := f.writeConfig(tmp); err != nil {
		return "", err
	}
	if err := f.writeTree(tmp); err != nil {
		return "", err
	}

	input, err := os.ReadFile(filepath.Join(f.dir, "input.json"))
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	input = []byte(strings.ReplaceAll(string(input), tmpMarker, tmp))
	getenv := func(k string) string { return env[k] }

	if f.isSubagent() {
		return subagentRows(subagent.Build(input, getenv))
	}
	return statusline.Build(input, getenv), nil
}

// optional returns the content of one of the fixture's optional files, and whether it exists.
func (f *Fixture) optional(name string) (string, bool) {
	root, err := os.OpenRoot(f.dir)
	if err != nil {
		return "", false
	}
	defer func() { _ = root.Close() }()
	data, err := root.ReadFile(name)
	return string(data), err == nil
}

func (f *Fixture) isSubagent() bool {
	data, _ := f.optional("command")
	return strings.TrimSpace(data) == subagentName
}

func (f *Fixture) readEnv(env map[string]string) {
	data, _ := f.optional("env")
	for line := range strings.SplitSeq(data, "\n") {
		if k, v, ok := strings.Cut(line, "="); ok && !strings.HasPrefix(k, "#") {
			env[k] = v
		}
	}
}

func (f *Fixture) writeConfig(tmp string) error {
	data, ok := f.optional("config")
	if !ok {
		return nil
	}
	appDir := filepath.Join(tmp, "claude-statusline")
	if err := os.MkdirAll(appDir, privateDir); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	text := strings.ReplaceAll(data, tmpMarker, tmp)
	if err := os.WriteFile(filepath.Join(appDir, "config"), []byte(text), privateFile); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// writeTree creates the files listed in the fixture's tree file: one path=content per line, a trailing / for a
// directory, a leading +x for an executable file, and \n in content for a newline.
func (f *Fixture) writeTree(tmp string) error {
	data, _ := f.optional("tree")
	for line := range strings.SplitSeq(data, "\n") {
		if line == "" {
			continue
		}
		if err := writeTreeEntry(tmp, line); err != nil {
			return err
		}
	}
	return nil
}

func writeTreeEntry(tmp, line string) error {
	mode := os.FileMode(privateFile)
	if rest, ok := strings.CutPrefix(line, "+x "); ok {
		line, mode = rest, executableFile
	}
	rel, content, _ := strings.Cut(line, "=")
	content = strings.ReplaceAll(content, `\n`, "\n")
	path := filepath.Join(tmp, rel)
	if strings.HasSuffix(rel, "/") {
		if err := os.MkdirAll(path, privateDir); err != nil {
			return fmt.Errorf("create %s: %w", rel, err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), privateDir); err != nil {
		return fmt.Errorf("create directory for %s: %w", rel, err)
	}
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(content, tmpMarker, tmp)+"\n"), mode); err != nil {
		return fmt.Errorf("write %s: %w", rel, err)
	}
	return nil
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

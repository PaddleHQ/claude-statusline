package featuretest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/featuretest/style"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
)

const (
	tmpMarker = "@TMP@"

	dirPerm  = 0o700
	filePerm = 0o600

	controlChars = "[\x00-\x09\x0b-\x1a\x1c-\x1f\x7f]"
)

var (
	errNoOutput    = errors.New("no output to check: a When step must run first")
	errPathEscapes = errors.New("path leaves the temporary directory")
	errDiffers     = errors.New("differs from what the scenario expects (- expected, + actual)")
)

type worldKey struct{}

// world is the state of one scenario: a private home directory, the environment, the session JSON the
// command reads and what it printed.
type world struct {
	suite   *Suite
	tmp     string
	env     map[string]string
	session string
	out     string
	styled  string
	screen  string
	ran     bool
}

func newWorld(s *Suite) (*world, error) {
	tmp, err := os.MkdirTemp("", "feature-")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	return &world{
		suite: s,
		tmp:   tmp,
		env: map[string]string{
			"HOME":              tmp,
			"CLAUDE_CONFIG_DIR": tmp,
		},
	}, nil
}

func worldOf(ctx context.Context) *world {
	w, _ := ctx.Value(worldKey{}).(*world)
	return w
}

func (w *world) close() error {
	if err := os.RemoveAll(w.tmp); err != nil {
		return fmt.Errorf("remove temp dir: %w", err)
	}
	return nil
}

func (w *world) getenv(name string) string {
	return w.env[name]
}

func (w *world) expand(text string) string {
	return strings.ReplaceAll(text, tmpMarker, w.tmp)
}

func (w *world) setSession(text string) {
	w.session = w.expand(text)
}

func (w *world) setEnv(name, value string) {
	w.env[name] = value
}

func (w *world) writeConfig(text string) error {
	return w.writeFile(filepath.Join("claude-statusline", "config"), text, filePerm)
}

func (w *world) writeCache(text string) error {
	return w.writeFile(w.cachePath(), text, filePerm)
}

func (w *world) cachePath() string {
	return filepath.Join("claude-statusline", "cache", "ratelimits")
}

// writeFile creates a file under the temporary directory holding text and a final newline.
func (w *world) writeFile(rel, text string, perm os.FileMode) error {
	path, err := w.path(rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("create directory for %s: %w", rel, err)
	}
	if err := os.WriteFile(path, []byte(w.expand(text)+"\n"), perm); err != nil {
		return fmt.Errorf("write %s: %w", rel, err)
	}
	return nil
}

func (w *world) makeDir(rel string) error {
	path, err := w.path(rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(path, dirPerm); err != nil {
		return fmt.Errorf("create %s: %w", rel, err)
	}
	return nil
}

func (w *world) path(rel string) (string, error) {
	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("%w: %s", errPathEscapes, rel)
	}
	return filepath.Join(w.tmp, rel), nil
}

func (w *world) renderStatusLine(ctx context.Context) {
	var out bytes.Buffer
	w.suite.Execute(nil, strings.NewReader(w.session), &out, w.getenv)

	w.ran = true
	w.screen = out.String()
	w.out = escapeESC(w.screen)
	w.styled = w.screen
	t := godog.T(ctx)
	assert.NotRegexp(t, controlChars, w.screen, "output contains a control character other than newline and escape")
	assert.NoFileExists(t, filepath.Join(w.tmp, "pwned"), "a config value was executed")
}

func (w *world) renderSubagentRows(ctx context.Context) error {
	var out bytes.Buffer
	w.suite.Execute([]string{"subagent"}, strings.NewReader(w.session), &out, w.getenv)

	var rows, styledRows, contents []string
	for line := range strings.SplitSeq(strings.TrimSuffix(out.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		var row struct {
			ID      json.RawMessage `json:"id"`
			Content string          `json:"content"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return fmt.Errorf("output line %q is not a row: %w", line, err)
		}
		contents = append(contents, row.Content)
		rows = append(rows, string(row.ID)+`\t`+escapeESC(row.Content))
		styledRows = append(styledRows, string(row.ID)+`\t`+row.Content)
	}

	w.ran = true
	w.screen = strings.Join(contents, "\n")
	w.out = strings.Join(rows, "\n")
	w.styled = strings.Join(styledRows, "\n")
	t := godog.T(ctx)
	assert.NotRegexp(t, controlChars, w.screen, "content contains a control character other than newline and escape")
	assert.NoFileExists(t, filepath.Join(w.tmp, "usage-latest.json"), "the subagent command must not write the usage snapshot")
	return nil
}

func (w *world) checkRawOutput(want string) error {
	if !w.ran {
		return errNoOutput
	}
	return compare("output", w.expand(want), w.out)
}

func (w *world) decodeOutput() (style.Block, error) {
	if !w.ran {
		return nil, errNoOutput
	}
	got, err := style.Decode(strings.ReplaceAll(w.styled, w.tmp, tmpMarker))
	if err != nil {
		return nil, fmt.Errorf("decode the output %q: %w", w.out, err)
	}
	return got, nil
}

func (w *world) checkOutput(want string) error {
	got, err := w.decodeOutput()
	if err != nil {
		return err
	}
	expected, err := style.Parse(want)
	if err != nil {
		return fmt.Errorf("read the expected output: %w", err)
	}
	if expected.Equal(got) {
		return nil
	}
	wantText, err := expected.Format()
	if err != nil {
		return fmt.Errorf("write the expected output: %w", err)
	}
	gotText, err := got.Format()
	if err != nil {
		return fmt.Errorf("write the actual output: %w", err)
	}
	return fmt.Errorf("%w\nthe actual output in the same format:\n%s", compare("output", wantText, gotText), gotText)
}

func (w *world) checkText(want string) error {
	got, err := w.decodeOutput()
	if err != nil {
		return err
	}
	return compare("text", trimRows(want), trimRows(got.Text()))
}

func trimRows(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

func (w *world) checkCache(want string) error {
	got, _ := os.ReadFile(filepath.Join(w.tmp, w.cachePath()))
	return compare("rate-limit cache", strings.TrimSpace(want), strings.TrimSpace(string(got)))
}

func escapeESC(text string) string {
	return strings.ReplaceAll(text, "\033", `\e`)
}

// compare returns nil when the texts match and otherwise an error showing them line by line.
func compare(what, want, got string) error {
	if want == got {
		return nil
	}
	wantLines, gotLines := strings.Split(want, "\n"), strings.Split(got, "\n")
	var diff strings.Builder
	for i := range max(len(wantLines), len(gotLines)) {
		switch {
		case i < len(wantLines) && i < len(gotLines) && wantLines[i] == gotLines[i]:
			diff.WriteString("  " + wantLines[i] + "\n")
		default:
			if i < len(wantLines) {
				diff.WriteString("- " + wantLines[i] + "\n")
			}
			if i < len(gotLines) {
				diff.WriteString("+ " + gotLines[i] + "\n")
			}
		}
	}
	return fmt.Errorf("%s %w:\n%s", what, errDiffers, diff.String())
}

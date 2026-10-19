package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/segments"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const reportNote = "The line is where the parser noticed the problem; the mistake may be just before it."

func writeConfig(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(text), 0o600))
	return path
}

func TestProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		text       string
		setup      func(t *testing.T) string
		wantNil    bool
		wantLine   int
		wantReason string
	}{
		{name: "missing file", setup: func(t *testing.T) string {
			t.Helper()
			return filepath.Join(t.TempDir(), "config.yaml")
		}, wantNil: true},
		{name: "no path", setup: func(*testing.T) string { return "" }, wantNil: true},
		{name: "empty file", text: "", wantNil: true},
		{name: "whitespace only", text: "  \n\n   \n", wantNil: true},
		{name: "comments only", text: "# a comment\n  # another\n", wantNil: true},
		{name: "null document", text: "~\n", wantNil: true},
		{name: "document marker only", text: "---\n", wantNil: true},
		{name: "unclosed flow sequence", text: "a: [1", wantLine: 1, wantReason: "did not find expected"},
		{name: "bad indentation", text: "hooks:\n  ttl: 60\n timeout: 2\n", wantLine: 2, wantReason: "did not find expected key"},
		{name: "a list", text: "- a\n- b\n", wantLine: 0, wantReason: "the top level must be a mapping"},
		{name: "a scalar", text: "just a string", wantLine: 0, wantReason: "the top level must be a mapping"},
		{name: "duplicate key", text: "a: 1\na: 2\n", wantLine: 2, wantReason: `mapping key "a" already defined at line 1`},
		{name: "control character", text: "a: 1\x00", wantLine: 0, wantReason: "control characters are not allowed"},
		{name: "unknown anchor", text: "a: *x", wantLine: 0, wantReason: "unknown anchor 'x' referenced"},
		{name: "tab indentation", text: "hooks:\n\tttl: 60\n", wantLine: 2, wantReason: "found character that cannot start any token"},
		{name: "too large", text: strings.Repeat("#", 1<<20+1), wantLine: 0, wantReason: "larger than 1 MiB"},
		{name: "a directory", setup: func(t *testing.T) string {
			t.Helper()
			path := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.Mkdir(path, 0o700))
			return path
		}, wantLine: 0, wantReason: "not a regular file"},
		{name: "a dangling symlink", setup: func(t *testing.T) string {
			t.Helper()
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			require.NoError(t, os.Symlink(filepath.Join(dir, "gone.yaml"), path))
			return path
		}, wantNil: true},
		{name: "a path through a file", setup: func(t *testing.T) string {
			t.Helper()
			file := filepath.Join(t.TempDir(), "file")
			require.NoError(t, os.WriteFile(file, nil, 0o600))
			return filepath.Join(file, "config.yaml")
		}, wantLine: 0, wantReason: "not a directory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var path string
			if tt.setup != nil {
				path = tt.setup(t)
			} else {
				path = writeConfig(t, tt.text)
			}
			p := config.Load(path).Problem()
			if tt.wantNil {
				assert.Nil(t, p)
				return
			}
			require.NotNil(t, p)
			assert.Equal(t, path, p.Path)
			assert.Equal(t, tt.wantLine, p.Line)
			assert.Contains(t, p.Reason, tt.wantReason)
			assert.NotContains(t, p.Reason, "\n")
		})
	}
}

func TestProblemKeepsOnlyTheFirstDuplicate(t *testing.T) {
	t.Parallel()

	p := config.New("a: 1\na: 2\nb: 1\nb: 2\n").Problem()
	require.NotNil(t, p)
	assert.Equal(t, 2, p.Line)
	assert.Equal(t, `mapping key "a" already defined at line 1`, p.Reason)
	assert.Empty(t, p.Path)
}

func TestProblemFIFO(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("mkfifo is unavailable: %v", err)
	}
	done := make(chan *config.Problem, 1)
	go func() { done <- config.Load(path).Problem() }()
	select {
	case p := <-done:
		require.NotNil(t, p)
		assert.Equal(t, 0, p.Line)
		assert.Equal(t, "not a regular file", p.Reason)
	case <-time.After(time.Second):
		require.FailNow(t, "Load blocked on a FIFO")
	}
}

func TestProblemUnreadable(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root can read any file")
	}
	path := writeConfig(t, "separator: x\n")
	require.NoError(t, os.Chmod(path, 0o000))
	p := config.Load(path).Problem()
	require.NotNil(t, p)
	assert.Equal(t, 0, p.Line)
	assert.Equal(t, "permission denied", p.Reason)
}

func TestProblemUntouchedDump(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, config.CommentedDefaults(segments.Defaults()...))
	assert.Nil(t, config.Load(path, segments.Defaults()...).Problem())
}

func TestProblemDoesNotChangeValues(t *testing.T) {
	t.Parallel()

	for _, text := range []string{
		"separator: \"x\"\nseparator: \"y\"\n",
		"separator: \"x\"\nhooks: [1\n",
		"separator: \"x\"\nhooks:\n  ttl: 5\n  ttl: 6\n",
	} {
		cfg := config.Load(writeConfig(t, text))
		require.NotNil(t, cfg.Problem(), "%q", text)
		assert.Equal(t, "·", cfg.String("separator"), "%q", text)
		assert.Equal(t, 60, cfg.Int("hooks.ttl"), "%q", text)
	}
}

func numbered(from, to int) string {
	var b strings.Builder
	for i := from; i <= to; i++ {
		fmt.Fprintf(&b, "k%d: %d\n", i, i)
	}
	return b.String()
}

func TestProblemReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "a line in the middle of the file",
			text: numbered(1, 11) + "k12: a: b\n" + numbered(13, 20),
			want: "12 mapping values are not allowed in this context\n" +
				"\n" +
				"   9 | k9: 9\n" +
				"  10 | k10: 10\n" +
				"  11 | k11: 11\n" +
				"> 12 | k12: a: b\n" +
				"  13 | k13: 13\n" +
				"  14 | k14: 14\n" +
				"\n" + reportNote,
		},
		{
			name: "the first line",
			text: "a: [1\nb: 2\nc: 3\nd: 4\n",
			want: "1 did not find expected ',' or ']'\n" +
				"\n" +
				"> 1 | a: [1\n" +
				"  2 | b: 2\n" +
				"  3 | c: 3\n" +
				"\n" + reportNote,
		},
		{
			name: "the last line",
			text: "a: 1\nb: a: b",
			want: "2 mapping values are not allowed in this context\n" +
				"\n" +
				"  1 | a: 1\n" +
				"> 2 | b: a: b\n" +
				"\n" + reportNote,
		},
		{
			name: "the empty line after the final break",
			text: "a: \"abc\nb: 1\n",
			want: "3 found unexpected end of stream\n" +
				"\n" +
				"  1 | a: \"abc\n" +
				"  2 | b: 1\n" +
				"> 3 |\n" +
				"\n" + reportNote,
		},
		{
			name: "line 0",
			text: "- a\n- b\n",
			want: "the top level must be a mapping",
		},
		{
			name: "a tab is shown, not dropped",
			text: "hooks:\n\tttl: 60\n",
			want: "2 found character that cannot start any token\n" +
				"\n" +
				"  1 | hooks:\n" +
				"> 2 | \u2192ttl: 60\n" +
				"  3 |\n" +
				"\n" + reportNote,
		},
		{
			name: "a file split by carriage returns",
			text: "a: 1\rb: a: b\rc: 3",
			want: "2 mapping values are not allowed in this context\n" +
				"\n" +
				"  1 | a: 1\n" +
				"> 2 | b: a: b\n" +
				"  3 | c: 3\n" +
				"\n" + reportNote,
		},
		{
			name: "a file split by CRLF",
			text: "a: 1\r\nb: a: b\r\nc: 3",
			want: "2 mapping values are not allowed in this context\n" +
				"\n" +
				"  1 | a: 1\n" +
				"> 2 | b: a: b\n" +
				"  3 | c: 3\n" +
				"\n" + reportNote,
		},
		{
			name: "a file split by line separators",
			text: "a: 1\u2028b: a: b\u2029c: 3\u0085d: 4",
			want: "2 mapping values are not allowed in this context\n" +
				"\n" +
				"  1 | a: 1\n" +
				"> 2 | b: a: b\n" +
				"  3 | c: 3\n" +
				"  4 | d: 4\n" +
				"\n" + reportNote,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := config.New(tt.text).Problem()
			require.NotNil(t, p)
			assert.Equal(t, tt.want, p.Report())
		})
	}
}

func TestProblemReportCleansText(t *testing.T) {
	t.Parallel()

	p := config.NewProblem(2, "bad\x1b[2J \u202ereason\n", "a: 1\x07\nb: \x1b[31mred\u202e\u200b\nc: 3")
	assert.Equal(t, "2 bad[2J reason\n"+
		"\n"+
		"  1 | a: 1\n"+
		"> 2 | b: [31mred\n"+
		"  3 | c: 3\n"+
		"\n"+reportNote, p.Report())
}

func TestProblemReportOutsideTheFile(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "9 reason", config.NewProblem(9, "reason", "a: 1\nb: 2").Report())
	assert.Equal(t, "3 reason\n\n  1 | a: 1\n  2 | b: 2\n> 3 |\n\n"+reportNote,
		config.NewProblem(3, "reason", "a: 1\nb: 2").Report())
}

func TestClean(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "abc[31mdef", config.Clean("a\x00b\tc\x1b[31m\u202ed\u200be\u2066f\u0085"))
	assert.Equal(t, "plain · text ✓", config.Clean("plain · text ✓"))
}

func TestUnknownKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want []string
	}{
		{name: "a top-level key from a section", text: "ttl: 60\n", want: []string{"ttl"}},
		{name: "a key under the wrong section", text: "cache:\n  warn: 5\n  width: 20\n", want: []string{"cache.width"}},
		{name: "a clean file", text: "separator: x\nhooks:\n  ttl: 5\ncache:\n  warn: 5\n", want: nil},
		{name: "several, sorted", text: "zz: 1\nhooks:\n  nope: 1\naa: 2\n", want: []string{"aa", "hooks.nope", "zz"}},
		{name: "a broken file", text: "ttl: 60\nttl: 61\n", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, config.New(tt.text, segments.Defaults()...).UnknownKeys(segments.Defaults()...))
		})
	}
}

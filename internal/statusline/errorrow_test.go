package statusline_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/statusline"
	"github.com/PaddleHQ/claude-statusline/internal/theme"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	rowHome = "/home/u"
	rowPath = "/home/u/.claude/claude-statusline/config.yaml"
	rowHint = " · run /statusline:config-check"
)

func plainTheme() *theme.Theme {
	th := theme.Load("classic", "", "")
	plain := th.WithoutColor()
	return &plain
}

func TestErrorRow(t *testing.T) {
	t.Parallel()

	keyProblem := &config.Problem{Path: rowPath, Line: 12, Reason: "did not find expected key"}
	tests := []struct {
		name    string
		problem *config.Problem
		columns int
		want    string
	}{
		{
			name:    "A wide terminal shows the whole row",
			problem: keyProblem,
			columns: 200,
			want:    "~/.claude/claude-statusline/config.yaml:12 did not find expected key" + rowHint,
		},
		{
			name:    "Columns of zero mean no limit",
			problem: keyProblem,
			columns: 0,
			want:    "~/.claude/claude-statusline/config.yaml:12 did not find expected key" + rowHint,
		},
		{
			name:    "A line of 0 names no line",
			problem: &config.Problem{Path: rowPath, Reason: "the top level must be a mapping"},
			columns: 200,
			want:    "~/.claude/claude-statusline/config.yaml: the top level must be a mapping" + rowHint,
		},
		{
			name:    "One column short loses the left of the path and keeps the hint",
			problem: keyProblem,
			columns: utf8.RuneCountInString("~/.claude/claude-statusline/config.yaml:12 did not find expected key"+rowHint) - 1,
			want:    "…/claude-statusline/config.yaml:12 did not find expected key" + rowHint,
		},
		{
			name:    "Narrower, the path is trimmed to the file and the hint goes",
			problem: keyProblem,
			columns: utf8.RuneCountInString("…/config.yaml:12 did not find expected key"),
			want:    "…/config.yaml:12 did not find expected key",
		},
		{
			name:    "Narrower again, the reason ends with an ellipsis and the line stays",
			problem: keyProblem,
			columns: 27,
			want:    "…/config.yaml:12 did not f…",
		},
		{
			name:    "A path outside the home directory is trimmed too",
			problem: &config.Problem{Path: "/etc/x/config.yaml", Line: 3, Reason: "bad"},
			columns: utf8.RuneCountInString("…/config.yaml:3 bad"),
			want:    "…/config.yaml:3 bad",
		},
		{
			name:    "Control characters, escapes and bidirectional marks never reach the row",
			problem: &config.Problem{Path: "/home/u/a\x1b[2J\u202eb/config.yaml", Line: 1, Reason: "bad \x1b[31mred\x1b[0m\nnext"},
			columns: 200,
			want:    "~/a[2Jb/config.yaml:1 bad [31mred[0mnext" + rowHint,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, statusline.ErrorRow(tt.problem, plainTheme(), rowHome, tt.columns))
		})
	}
}

func TestErrorRowNeverExceedsTheColumns(t *testing.T) {
	t.Parallel()

	p := &config.Problem{Path: rowPath, Line: 12, Reason: "did not find expected key"}
	for _, columns := range []int{1, 2, 5, 10, 17, 18, 40, 60, 100} {
		got := statusline.ErrorRow(p, plainTheme(), rowHome, columns)
		assert.LessOrEqual(t, layout.Width(got), columns, "columns %d: %q", columns, got)
		assert.NotEmpty(t, got)
	}
}

func TestErrorRowWithoutAHomeLeavesThePathAlone(t *testing.T) {
	t.Parallel()

	p := &config.Problem{Path: rowPath, Line: 1, Reason: "bad"}
	got := statusline.ErrorRow(p, plainTheme(), "", 0)
	assert.True(t, strings.HasPrefix(got, rowPath+":1 bad"), got)
}

func TestErrorRowIsPainted(t *testing.T) {
	t.Parallel()

	th := theme.Load("classic", "", "")
	require.NotEmpty(t, th.BadColor)

	got := statusline.ErrorRow(&config.Problem{Path: rowPath, Line: 2, Reason: "bad"}, &th, rowHome, 200)
	assert.True(t, strings.HasPrefix(got, th.BadColor), got)
	assert.True(t, strings.HasSuffix(got, th.Reset), got)
}

var controlChars = regexp.MustCompile(`[\x00-\x1f\x7f-\x9f]`)

func FuzzErrorRow(f *testing.F) {
	f.Add(rowPath, 12, "did not find expected key", rowHome, 80)
	f.Add(rowPath, 0, "the top level must be a mapping", rowHome, 10)
	f.Add("a\x1b[2J\u202eb", 1, "x\ny\x1b[31m", "", 1)
	f.Add("", 0, "", "", 5)
	f.Add("/", -3, "…", "/", 3)

	th := plainTheme()
	f.Fuzz(func(t *testing.T, path string, line int, reason, home string, columns int) {
		if columns < 1 {
			columns = 1
		}
		got := statusline.ErrorRow(&config.Problem{Path: path, Line: line, Reason: reason}, th, home, columns)
		assert.False(t, controlChars.MatchString(got), "control character in %q", got)
		assert.NotContains(t, got, "\n")
		assert.LessOrEqual(t, layout.Width(got), columns)
	})
}

func TestBuildShowsErrorRow(t *testing.T) {
	t.Parallel()

	const session = `{"model":{"display_name":"Opus 4.7"}}`
	build := func(t *testing.T, config, input string) string {
		t.Helper()

		dir := t.TempDir()
		if config != "" {
			path := filepath.Join(dir, "claude-statusline", "config.yaml")
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
			require.NoError(t, os.WriteFile(path, []byte(config), 0o600))
		}
		vars := map[string]string{"HOME": dir, "CLAUDE_CONFIG_DIR": dir, "NO_COLOR": "1", "STATUSLINE_NOW": "1760000000"}
		return statusline.Build([]byte(input), func(k string) string { return vars[k] })
	}
	plain := build(t, "", session)
	require.NotEmpty(t, plain)
	row := "~/claude-statusline/config.yaml:1 did not find expected ',' or ']'" + rowHint

	t.Run("A broken config gives the error row above an unchanged default line", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, row+"\n"+plain, build(t, "segments: [model", session))
	})
	t.Run("With nothing to show the output is the row alone, with no newline", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, row, build(t, "segments: [model", "not json"))
	})
	for name, config := range map[string]string{
		"A valid config":    "separator: \"|\"",
		"A missing config":  "",
		"A null document":   "~",
		"Only comments":     "# nothing here",
		"A document marker": "---",
	} {
		t.Run(name+" adds no row", func(t *testing.T) {
			t.Parallel()

			assert.NotContains(t, build(t, config, session), "config-check")
		})
	}
	t.Run("An empty HOME and CLAUDE_CONFIG_DIR add no row and do not panic", func(t *testing.T) {
		t.Parallel()

		got := statusline.Build([]byte(session), func(k string) string { return map[string]string{"NO_COLOR": "1"}[k] })
		assert.NotContains(t, got, "config-check")
	})
}

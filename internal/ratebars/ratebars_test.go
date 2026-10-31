package ratebars_test

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/ratebars"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	now     = "1760000000"
	cache   = `{"five_hour":{"used":75,"resets_at":1760009000},"seven_day":{"used":40,"resets_at":1760302400}}`
	escapes = "\x1b["
)

var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

type window struct {
	OK      bool              `json:"ok"`
	Percent string            `json:"percent"`
	Bars    map[string]string `json:"bars"`
}

type result struct {
	Rate5h window `json:"rate5h"`
	Rate7d window `json:"rate7d"`
}

type env struct {
	cache  string
	config string
	vars   map[string]string
}

func (e *env) run(t *testing.T, args ...string) (string, error) {
	t.Helper()

	dir := t.TempDir()
	vars := map[string]string{"HOME": dir, "CLAUDE_CONFIG_DIR": dir, "STATUSLINE_NOW": now}
	maps.Copy(vars, e.vars)
	if e.cache != "" {
		path := filepath.Join(dir, "claude-statusline", "cache", "ratelimits")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(e.cache+"\n"), 0o600))
	}
	if e.config != "" {
		path := filepath.Join(dir, "claude-statusline", "config.yaml")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(e.config+"\n"), 0o600))
	}
	return ratebars.Build(args, func(k string) string { return vars[k] })
}

func decode(t *testing.T, out string) result {
	t.Helper()

	require.Equal(t, byte('\n'), out[len(out)-1], "one line ends in a newline")
	var r result
	require.NoError(t, json.Unmarshal([]byte(out), &r))
	return r
}

func TestBuildHasOneBarPerCellCountAtThatWidth(t *testing.T) {
	t.Parallel()

	out, err := (&env{cache: cache}).run(t, "--min-cells=3", "--max-cells=60")
	require.NoError(t, err)
	r := decode(t, out)

	for _, w := range []window{r.Rate5h, r.Rate7d} {
		assert.True(t, w.OK)
		assert.Len(t, w.Bars, 58)
		for cells := 3; cells <= 60; cells++ {
			key := strconv.Itoa(cells)
			require.Contains(t, w.Bars, key)
			assert.Equal(t, cells, layout.Width(w.Bars[key]), "bar %s", key)
		}
	}
	assert.Equal(t, "75%", stripped(r.Rate5h.Percent))
	assert.Equal(t, "40%", stripped(r.Rate7d.Percent))
}

func TestBuildKeepsTheBarsInNumericOrder(t *testing.T) {
	t.Parallel()

	out, err := (&env{cache: cache, vars: map[string]string{"NO_COLOR": "1"}}).run(t, "--min-cells=9", "--max-cells=11")
	require.NoError(t, err)
	assert.Regexp(t, `"bars":\{"9":"[^"]*","10":"[^"]*","11":"[^"]*"\}`, out)
}

func TestBuildMatchesTheSegmentsPacing(t *testing.T) {
	t.Parallel()

	// 75% used with half the 5h window elapsed is 25 points off the pace: amber on the classic theme.
	out, err := (&env{cache: cache}).run(t, "--min-cells=10", "--max-cells=10")
	require.NoError(t, err)
	r := decode(t, out)
	assert.Contains(t, r.Rate5h.Bars["10"], "\x1b[33m", "amber fill")
	assert.Contains(t, r.Rate5h.Bars["10"], "┃", "marker at the pace")
	assert.Equal(t, "75%", r.Rate5h.Percent, "a solid theme leaves the percentage uncolored")
}

func TestBuildHonoursThresholdsFromTheConfig(t *testing.T) {
	t.Parallel()

	out, err := (&env{cache: cache, config: "rate5h:\n  warn: 30\n  bad: 60"}).run(t, "--min-cells=10", "--max-cells=10")
	require.NoError(t, err)
	assert.Contains(t, decode(t, out).Rate5h.Bars["10"], "\x1b[32m", "within the raised warn threshold")
}

func TestBuildGradientThemeDiffersFromClassic(t *testing.T) {
	t.Parallel()

	classic, err := (&env{cache: cache}).run(t, "--min-cells=8", "--max-cells=8")
	require.NoError(t, err)
	gradient, err := (&env{cache: cache, config: "theme: gradient"}).run(t, "--min-cells=8", "--max-cells=8")
	require.NoError(t, err)

	c, g := decode(t, classic), decode(t, gradient)
	assert.NotEqual(t, c.Rate5h.Bars["8"], g.Rate5h.Bars["8"])
	assert.Contains(t, g.Rate5h.Bars["8"], "▰")
	assert.Contains(t, g.Rate5h.Percent, "\x1b[33m", "the gradient theme moves the pace color to the percentage")
	assert.Equal(t, 8, layout.Width(g.Rate5h.Bars["8"]))
}

func TestBuildNoColorHasNoEscapes(t *testing.T) {
	t.Parallel()

	out, err := (&env{cache: cache, vars: map[string]string{"NO_COLOR": "1"}}).run(t, "--min-cells=4", "--max-cells=6")
	require.NoError(t, err)
	assert.NotContains(t, out, `\u001b`)
	r := decode(t, out)
	assert.NotContains(t, r.Rate5h.Bars["5"], escapes)
	assert.Equal(t, "75%", r.Rate5h.Percent)
}

func TestBuildWithoutAReadingIsNotOK(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		cache string
	}{
		{"no cache file", ""},
		{"corrupt cache file", "{"},
		{"window already reset", `{"five_hour":{"used":75,"resets_at":1760000000},"seven_day":{"used":40,"resets_at":1759999999}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out, err := (&env{cache: tt.cache}).run(t, "--min-cells=3", "--max-cells=5")
			require.NoError(t, err)
			r := decode(t, out)
			for _, w := range []window{r.Rate5h, r.Rate7d} {
				assert.False(t, w.OK)
				assert.Equal(t, "\x1b[2m--\x1b[0m", w.Percent, "muted")
				require.Len(t, w.Bars, 3)
				assert.Equal(t, "\x1b[2m░░░░\x1b[0m", w.Bars["4"], "empty glyphs in the muted color, no marker")
			}
		})
	}
}

func TestBuildOneWindowMissing(t *testing.T) {
	t.Parallel()

	out, err := (&env{cache: `{"five_hour":{"used":10,"resets_at":1760009000}}`}).run(t, "--min-cells=4", "--max-cells=4")
	require.NoError(t, err)
	r := decode(t, out)
	assert.True(t, r.Rate5h.OK)
	assert.False(t, r.Rate7d.OK)
}

func TestBuildAbsentBarsUseTheGradientEmptyColor(t *testing.T) {
	t.Parallel()

	out, err := (&env{config: "theme: gradient"}).run(t, "--min-cells=3", "--max-cells=3")
	require.NoError(t, err)
	r := decode(t, out)
	assert.Equal(t, "\x1b[38;2;74;74;74m▱▱▱\x1b[0m", r.Rate5h.Bars["3"])
}

func TestBuildDoesNotWriteTheCache(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	vars := map[string]string{"HOME": dir, "CLAUDE_CONFIG_DIR": dir, "STATUSLINE_NOW": now}
	_, err := ratebars.Build([]string{"--min-cells=3", "--max-cells=4"}, func(k string) string { return vars[k] })
	require.NoError(t, err)
	assert.NoDirExists(t, filepath.Join(dir, "claude-statusline"))
}

func TestBuildRejectsBadFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{"no flags", nil},
		{"missing max", []string{"--min-cells=3"}},
		{"missing min", []string{"--max-cells=3"}},
		{"min below one", []string{"--min-cells=0", "--max-cells=3"}},
		{"max above 200", []string{"--min-cells=1", "--max-cells=201"}},
		{"min above max", []string{"--min-cells=5", "--max-cells=4"}},
		{"not a number", []string{"--min-cells=x", "--max-cells=4"}},
		{"unknown flag", []string{"--min-cells=1", "--max-cells=4", "--wide"}},
		{"stray argument", []string{"--min-cells=1", "--max-cells=4", "extra"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out, err := (&env{cache: cache}).run(t, tt.args...)
			require.ErrorIs(t, err, ratebars.ErrUsage)
			assert.Empty(t, out)
		})
	}
}

func TestBuildAcceptsTheBounds(t *testing.T) {
	t.Parallel()

	out, err := (&env{cache: cache}).run(t, "--min-cells", "1", "--max-cells", "200")
	require.NoError(t, err)
	assert.Len(t, decode(t, out).Rate5h.Bars, 200)
}

func stripped(s string) string { return sgr.ReplaceAllString(s, "") }

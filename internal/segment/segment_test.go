package segment_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/segment"
	"github.com/PaddleHQ/claude-statusline/internal/theme"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryNamesMatch(t *testing.T) {
	t.Parallel()

	for name, s := range segment.Registry {
		assert.Equal(t, name, s.Name(), "segment registered as %q", name)
	}
}

func plainClassic() theme.Theme {
	classic := theme.Load("classic", "", "")
	return classic.WithoutColor()
}

func renderOne(t *testing.T, name, json, cfg string, th theme.Theme) string {
	t.Helper()
	s, ok := segment.Lookup(name)
	require.True(t, ok, "%s is not registered", name)
	return s.Render(&segment.Context{Payload: payload.Parse([]byte(json)), Config: config.New(cfg).Section(""), Theme: th, Columns: 999, Home: "/home/me"})
}

func TestDisplayPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dir      string
		home     string
		prefixes []string
		want     string
	}{
		{"under home", "/home/me/work/app", "/home/me", nil, "~/work/app"},
		{"home itself", "/home/me", "/home/me", nil, "~"},
		{"home with a trailing slash", "/home/me/app", "/home/me/", nil, "~/app"},
		{"sibling of home is not under it", "/home/mellow/app", "/home/me", nil, "/home/mellow/app"},
		{"outside home", "/srv/app", "/home/me", nil, "/srv/app"},
		{"no home", "/home/me/app", "", nil, "/home/me/app"},
		{"tilde prefix", "/home/me/code/app", "/home/me", []string{"~/code"}, "app"},
		{"absolute prefix", "/srv/work/app", "/home/me", []string{"/srv/work"}, "app"},
		{"prefix with a trailing slash", "/srv/work/app", "/home/me", []string{"/srv/work/"}, "app"},
		{"longest prefix wins", "/home/me/code/team/app", "/home/me", []string{"~/code", "~/code/team"}, "app"},
		{"longest prefix wins in any order", "/home/me/code/team/app", "/home/me", []string{"~/code/team", "~/code"}, "app"},
		{"prefix needs a path boundary", "/home/me/codebase/app", "/home/me", []string{"~/code"}, "~/codebase/app"},
		{"prefix equal to the directory is not stripped", "/home/me/code", "/home/me", []string{"~/code"}, "~/code"},
		{"tilde prefix without home is ignored", "/home/me/code/app", "", []string{"~/code"}, "/home/me/code/app"},
		{"tilde user form is not expanded", "/home/me/code/app", "/home/me", []string{"~me/code"}, "~/code/app"},
		{"root prefix is ignored", "/srv/app", "", []string{"/"}, "/srv/app"},
		{"unmatched prefixes fall back to home", "/home/me/work/app", "/home/me", []string{"~/code", "/srv"}, "~/work/app"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, segment.DisplayPath(tt.dir, tt.home, tt.prefixes))
		})
	}
}

func TestPathRender(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".git"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o600))
	hostile := filepath.Join(tmp, "hostile")
	require.NoError(t, os.MkdirAll(filepath.Join(hostile, ".git"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(hostile, ".git", "HEAD"), []byte("ref: refs/heads/a\x1b[31mb\n"), 0o600))

	plain := plainClassic()
	jsonFor := func(dir string) string { return `{"workspace":{"current_dir":"` + dir + `"}}` }

	tests := []struct {
		name string
		json string
		cfg  string
		th   theme.Theme
		want string
	}{
		{"home abbreviated", `{"cwd":"/home/me/x"}`, "", plain, "~/x"},
		{"branch appended", jsonFor(repo), "path.strip = " + tmp, plain, "repo (main)"},
		{"branch can be turned off", jsonFor(repo), "path.strip = " + tmp + "\npath.branch = false", plain, "repo"},
		{"branch is sanitized", jsonFor(hostile), "path.strip = " + tmp, plain, "hostile (a[31mb)"},
		{"painted with the path color", `{"cwd":"/srv/x"}`, "", theme.Theme{Path: "<p>", Reset: "</p>"}, "<p>/srv/x</p>"},
		{"no directory hides the segment", `{}`, "", plain, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, renderOne(t, "path", tt.json, tt.cfg, tt.th))
		})
	}
}

func TestCtxRender(t *testing.T) {
	t.Parallel()

	markers := theme.Theme{Good: "<g>", Warn: "<w>", Bad: "<b>", Muted: "<d>", Reset: "</>", Full: "F", Partials: [7]string{"1", "2", "3", "4", "5", "6", "7"}, Empty: "E"}
	tests := []struct {
		name string
		json string
		cfg  string
		want string
	}{
		{"good below warn", `{"context_window":{"used_percentage":49}}`, "ctx.width = 4", "ctx: <g>F</><g>F</><d>E</><d>E</> 49%"},
		{"warn at the threshold", `{"context_window":{"used_percentage":50}}`, "ctx.width = 4", "ctx: <w>F</><w>F</><d>E</><d>E</> 50%"},
		{"bad at the threshold", `{"context_window":{"used_percentage":90}}`, "ctx.width = 4", "ctx: <b>F</><b>F</><b>F</><b>5</> 90%"},
		{"custom thresholds", `{"context_window":{"used_percentage":20}}`, "ctx.width = 4\nctx.warn = 10\nctx.bad = 20", "ctx: <b>6</><d>E</><d>E</><d>E</> 20%"},
		{"percentage rounds to even", `{"context_window":{"used_percentage":42.5}}`, "ctx.width = 4", "ctx: <g>F</><g>5</><d>E</><d>E</> 42%"},
		{"zero is shown", `{"context_window":{"used_percentage":0}}`, "ctx.width = 4", "ctx: <d>E</><d>E</><d>E</><d>E</> 0%"},
		{"absent hides the segment", `{"context_window":{}}`, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, renderOne(t, "ctx", tt.json, tt.cfg, markers))
		})
	}
}

func TestFormatTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0k"},
		{1049, "1.0k"},
		{1500, "1.5k"},
		{999999, "999.9k"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{12345678, "12.3M"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, segment.FormatTokens(tt.n))
		})
	}
}

func TestTokensRender(t *testing.T) {
	t.Parallel()

	painted := theme.Theme{Text: "<t>", Muted: "<m>", Reset: "</>"}
	tests := []struct {
		name string
		json string
		cfg  string
		th   theme.Theme
		want string
	}{
		{"both", `{"context_window":{"total_input_tokens":1500,"total_output_tokens":250}}`, "", painted, "<t>↓ 1.5k</> <m>·</> <t>↑ 250</>"},
		{"custom separator", `{"context_window":{"total_input_tokens":1,"total_output_tokens":2}}`, "separator = |", painted, "<t>↓ 1</> <m>|</> <t>↑ 2</>"},
		{"input only", `{"context_window":{"total_input_tokens":1500}}`, "", painted, "<t>↓ 1.5k</>"},
		{"output only", `{"context_window":{"total_output_tokens":7}}`, "", painted, "<t>↑ 7</>"},
		{"neither hides the segment", `{"context_window":{"used_percentage":5}}`, "", painted, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, renderOne(t, "tokens", tt.json, tt.cfg, tt.th))
		})
	}
}

func TestCacheRender(t *testing.T) {
	t.Parallel()

	painted := theme.Theme{Text: "<t>", Bad: "<b>", Reset: "</>"}
	usage := func(in, create, read string) string {
		return `{"context_window":{"current_usage":{"input_tokens":` + in + `,"cache_creation_input_tokens":` + create + `,"cache_read_input_tokens":` + read + `}}}`
	}
	tests := []struct {
		name string
		json string
		cfg  string
		want string
	}{
		{"healthy", usage("10", "10", "80"), "", "<t>⧉ 80%</>"},
		{"at the threshold is not red", usage("95", "0", "5"), "", "<t>⧉ 5%</>"},
		{"below the threshold is red", usage("96", "0", "4"), "", "<b>⧉ 4%</>"},
		{"integer division truncates", usage("1", "0", "2"), "cache.warn = 0", "<t>⧉ 66%</>"},
		{"custom threshold", usage("10", "10", "80"), "cache.warn = 90", "<b>⧉ 80%</>"},
		{"all reads is 100", usage("0", "0", "10"), "", "<t>⧉ 100%</>"},
		{"zero denominator hides the segment", usage("0", "0", "0"), "", ""},
		{"no usage hides the segment", `{"context_window":{}}`, "", ""},
		{"missing counts are zero", `{"context_window":{"current_usage":{"cache_read_input_tokens":5}}}`, "", "<t>⧉ 100%</>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, renderOne(t, "cache", tt.json, tt.cfg, painted))
		})
	}
}

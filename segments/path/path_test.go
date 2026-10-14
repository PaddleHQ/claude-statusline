package path_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PaddleHQ/claude-statusline/internal/config"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
	"github.com/PaddleHQ/claude-statusline/internal/theme"
	"github.com/PaddleHQ/claude-statusline/segments/path"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func plainClassic() theme.Theme {
	classic := theme.Load("classic", "", "")
	return classic.WithoutColor()
}

func renderOne(json, cfg string, th theme.Theme) string {
	providers := plugin.NewProviders(&plugin.Sources{
		Config:  config.New(cfg, path.Plugin().Defaults),
		Theme:   th,
		Columns: 999,
		Home:    "/home/me",
	})
	return path.New(providers.For("path")).Render(&plugin.Session{Payload: payload.Parse([]byte(json))})
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
			assert.Equal(t, tt.want, path.DisplayPath(tt.dir, tt.home, tt.prefixes))
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
			assert.Equal(t, tt.want, renderOne(tt.json, tt.cfg, tt.th))
		})
	}
}

func TestHelp(t *testing.T) {
	t.Parallel()

	help := func(cfg string) string {
		providers := plugin.NewProviders(&plugin.Sources{Config: config.New(cfg, path.Plugin().Defaults)})
		return path.New(providers.For("path")).Help()
	}
	assert.Contains(t, help(""), "branch")
	assert.NotContains(t, help("path.branch = false"), "branch")
	assert.Contains(t, help("path.strip = ~/code /srv"), "~/code /srv")
}

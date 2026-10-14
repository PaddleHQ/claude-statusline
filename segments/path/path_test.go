package path_test

import (
	"testing"

	"github.com/PaddleHQ/claude-statusline/segments/path"
	"github.com/PaddleHQ/claude-statusline/segments/path/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

func build(t *testing.T, cwd, strip string, branch bool, branches map[string]string) *path.Path {
	t.Helper()

	sess := mocks.NewSession(t)
	sess.EXPECT().Cwd().Return(cwd).Maybe()
	cfg := mocks.NewConfig(t)
	cfg.EXPECT().String("strip").Return(strip).Maybe()
	cfg.EXPECT().Bool("branch").Return(branch).Maybe()
	paint := mocks.NewPainter(t)
	paint.EXPECT().Path(mock.Anything).RunAndReturn(func(s string) string { return "<p>" + s + "</>" }).Maybe()
	git := mocks.NewGit(t)
	git.EXPECT().Branch(mock.Anything).RunAndReturn(func(dir string) string { return branches[dir] }).Maybe()
	return path.New(path.Deps{Session: sess, Config: cfg, Paint: paint, Git: git, Home: "/home/me"})
}

func TestRender(t *testing.T) {
	t.Parallel()

	branches := map[string]string{"/home/me/repo": "main"}
	tests := []struct {
		name   string
		cwd    string
		strip  string
		branch bool
		want   string
	}{
		{"home abbreviated", "/home/me/x", "", true, "<p>~/x</>"},
		{"branch appended", "/home/me/repo", "", true, "<p>~/repo (main)</>"},
		{"branch can be turned off", "/home/me/repo", "", false, "<p>~/repo</>"},
		{"strip applies", "/home/me/repo", "~", true, "<p>repo (main)</>"},
		{"directory without a branch", "/srv/x", "", true, "<p>/srv/x</>"},
		{"no directory hides the segment", "", "", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, build(t, tt.cwd, tt.strip, tt.branch, branches).Render())
		})
	}
}

func TestNameAndKeys(t *testing.T) {
	t.Parallel()

	seg := build(t, "", "", true, nil)
	assert.Equal(t, "path", seg.Name())
	assert.Equal(t, []string{"strip", "branch"}, seg.Keys())
}

func TestHelp(t *testing.T) {
	t.Parallel()

	assert.Contains(t, build(t, "", "", true, nil).Help(), "branch")
	assert.NotContains(t, build(t, "", "", false, nil).Help(), "branch")
	assert.Contains(t, build(t, "", "~/code /srv", false, nil).Help(), "~/code /srv")
}

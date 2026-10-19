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
		{"the first listed prefix that matches is used", "/home/me/code/team/app", "/home/me", []string{"~/projects", "~/code"}, "team/app"},
		{"unmatched prefixes fall back to home", "/home/me/work/app", "/home/me", []string{"~/code", "/srv"}, "~/work/app"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, path.DisplayPath(tt.dir, tt.home, tt.prefixes))
		})
	}
}

type setup struct {
	cwd       string
	strip     []string
	branch    bool
	worktrees []string
	emoji     bool
	// checkout is what the reader reports for cwd: branch, worktree root, main root, project.
	checkout [4]string
}

func build(t *testing.T, s setup, wantCheckout bool) *path.Path {
	t.Helper()

	sess := mocks.NewSession(t)
	sess.EXPECT().WorkingDir().Return(s.cwd).Maybe()
	cfg := mocks.NewConfig(t)
	cfg.EXPECT().Strings("strip").Return(s.strip).Maybe()
	cfg.EXPECT().Strings("worktrees").Return(s.worktrees).Maybe()
	cfg.EXPECT().Bool("branch").Return(s.branch).Maybe()
	cfg.EXPECT().Bool("emoji").Return(s.emoji).Maybe()
	paint := mocks.NewPainter(t)
	paint.EXPECT().Path(mock.Anything).RunAndReturn(func(s string) string { return "<p>" + s + "</>" }).Maybe()
	paint.EXPECT().Branch(mock.Anything).RunAndReturn(func(s string) string { return "<b>" + s + "</>" }).Maybe()
	paint.EXPECT().Worktree(mock.Anything).RunAndReturn(func(s string) string { return "<w>" + s + "</>" }).Maybe()
	paint.EXPECT().Glyphs().Return("B", "W").Maybe()
	git := mocks.NewGit(t)
	if wantCheckout {
		git.EXPECT().Checkout(s.cwd).Return(s.checkout[0], s.checkout[1], s.checkout[2], s.checkout[3]).Once()
	}
	return path.New(path.Deps{Session: sess, Config: cfg, Paint: paint, Git: git, Home: "/home/me"})
}

func TestRender(t *testing.T) {
	t.Parallel()

	const (
		repo = "/home/me/repo"
		wt   = "/home/me/repo/.worktrees/wt"
	)
	wtCheckout := [4]string{"feat", wt, repo, "repo"}
	tests := []struct {
		name string
		s    setup
		want string
	}{
		{"home abbreviated", setup{cwd: "/home/me/x", branch: true}, "<p>~/x</>"},
		{"regular repository", setup{cwd: repo, branch: true, checkout: [4]string{"main"}}, "<p>~/repo (</><b>main</><p>)</>"},
		{"regular repository with emoji", setup{cwd: repo, branch: true, emoji: true, checkout: [4]string{"main"}}, "<p>~/repo (</><b>B main</><p>)</>"},
		{"branch can be turned off", setup{cwd: repo}, "<p>~/repo</>"},
		{"strip applies", setup{cwd: repo, strip: []string{"~"}, branch: true, checkout: [4]string{"main"}}, "<p>repo (</><b>main</><p>)</>"},
		{"directory without a branch", setup{cwd: "/srv/x", branch: true}, "<p>/srv/x</>"},
		{"no directory hides the segment", setup{branch: true}, ""},

		{"worktree", setup{cwd: wt, branch: true, worktrees: []string{".worktrees"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"worktree with emoji", setup{cwd: wt, branch: true, emoji: true, worktrees: []string{".worktrees"}, checkout: wtCheckout}, "<p>repo (</><w>W feat</><p>)</>"},
		{"worktree subdirectory keeps what follows the root", setup{cwd: wt + "/pkg/sub", branch: true, worktrees: []string{".worktrees"}, checkout: wtCheckout}, "<p>repo/pkg/sub (</><w>feat</><p>)</>"},
		{"strip is ignored for a worktree", setup{cwd: wt, branch: true, strip: []string{"~"}, worktrees: []string{".worktrees"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"worktree without a branch name", setup{cwd: wt, branch: true, worktrees: []string{".worktrees"}, checkout: [4]string{"", wt, repo, "repo"}}, "<p>repo</>"},
		{"branch off shows the project alone", setup{cwd: wt, emoji: true, worktrees: []string{".worktrees"}, checkout: wtCheckout}, "<p>repo</>"},
		{"absolute entry without a main root", setup{cwd: wt, branch: true, worktrees: []string{"/home/me/repo/.worktrees"}, checkout: [4]string{"feat", wt, "", "repo"}}, "<p>repo (</><w>feat</><p>)</>"},
		{"relative entry without a main root", setup{cwd: wt, branch: true, worktrees: []string{".worktrees"}, checkout: [4]string{"feat", wt, "", "repo"}}, "<p>~/repo/.worktrees/wt (</><b>feat</><p>)</>"},
		{"tilde entry", setup{cwd: wt, branch: true, worktrees: []string{"~/repo/.worktrees"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"tilde user entry is relative", setup{cwd: wt, branch: true, worktrees: []string{"~me/repo/.worktrees"}, checkout: wtCheckout}, "<p>~/repo/.worktrees/wt (</><b>feat</><p>)</>"},
		{"entry that cleans to the main root", setup{cwd: wt, branch: true, worktrees: []string{"./"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"entry with a trailing slash", setup{cwd: wt, branch: true, worktrees: []string{".worktrees/"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"entry with dot segments", setup{cwd: wt, branch: true, worktrees: []string{"a/../.worktrees"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"root entry is ignored", setup{cwd: wt, branch: true, worktrees: []string{"/"}, checkout: wtCheckout}, "<p>~/repo/.worktrees/wt (</><b>feat</><p>)</>"},
		{"any listed entry may match", setup{cwd: wt, branch: true, worktrees: []string{".claude/worktrees", ".worktrees"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"main repository itself is regular", setup{cwd: repo, branch: true, worktrees: []string{".worktrees"}, checkout: [4]string{"main"}}, "<p>~/repo (</><b>main</><p>)</>"},
		{"root equal to the entry is regular", setup{cwd: "/home/me/repo/.worktrees", branch: true, worktrees: []string{".worktrees"}, checkout: [4]string{"feat", "/home/me/repo/.worktrees", repo, "repo"}}, "<p>~/repo/.worktrees (</><b>feat</><p>)</>"},
		{"sibling named with the entry as a prefix is regular", setup{cwd: repo + "/.worktrees-old/wt", branch: true, worktrees: []string{".worktrees"}, checkout: [4]string{"feat", repo + "/.worktrees-old/wt", repo, "repo"}}, "<p>~/repo/.worktrees-old/wt (</><b>feat</><p>)</>"},
		{"directory outside the reported root is regular", setup{cwd: wt + "x", branch: true, worktrees: []string{".worktrees"}, checkout: wtCheckout}, "<p>~/repo/.worktrees/wtx (</><b>feat</><p>)</>"},
		{"worktree without a project is regular", setup{cwd: wt, branch: true, worktrees: []string{".worktrees"}, checkout: [4]string{"feat", wt, repo, ""}}, "<p>~/repo/.worktrees/wt (</><b>feat</><p>)</>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wantCheckout := tt.s.cwd != "" && (tt.s.branch || len(tt.s.worktrees) > 0)
			assert.Equal(t, tt.want, build(t, tt.s, wantCheckout).Render())
		})
	}
}

func TestGitIsNotReadWhenNothingNeedsIt(t *testing.T) {
	t.Parallel()

	got := build(t, setup{cwd: "/home/me/repo", checkout: [4]string{"main"}}, false).Render()
	assert.Equal(t, "<p>~/repo</>", got)
}

func TestNameAndKeys(t *testing.T) {
	t.Parallel()

	seg := build(t, setup{branch: true}, false)
	assert.Equal(t, "path", seg.Name())
	assert.Equal(t, []string{"strip", "branch", "worktrees", "emoji"}, seg.Keys())
}

func TestHelp(t *testing.T) {
	t.Parallel()

	assert.Contains(t, build(t, setup{branch: true}, false).Help(), "branch")
	assert.NotContains(t, build(t, setup{}, false).Help(), "branch")
	assert.Contains(t, build(t, setup{strip: []string{"~/code", "/my srv"}}, false).Help(), "~/code, /my srv")
	assert.Contains(t, build(t, setup{worktrees: []string{".worktrees", "~/wt"}}, false).Help(), "under .worktrees, ~/wt")
	assert.NotContains(t, build(t, setup{}, false).Help(), "worktree")
}

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

// checkout is what the git reader reports for the working directory.
type checkout struct {
	branch, root, mainRoot, project string
}

type setup struct {
	cwd       string
	strip     []string
	branch    bool
	worktrees []string
	emoji     bool
	checkout  checkout
}

// build lets the git reader be called either way, so a test of what is drawn does not restate when git is read.
func build(t *testing.T, s setup) *path.Path {
	t.Helper()

	git := mocks.NewGit(t)
	git.EXPECT().Checkout(s.cwd).Return(s.checkout.branch, s.checkout.root, s.checkout.mainRoot, s.checkout.project).Maybe()
	git.EXPECT().Branch(s.cwd).Return(s.checkout.branch).Maybe()
	return newPath(t, s, git)
}

// buildStrict expects exactly one git read when wantGit is set: Checkout when worktrees are listed, else Branch.
func buildStrict(t *testing.T, s setup, wantGit bool) *path.Path {
	t.Helper()

	git := mocks.NewGit(t)
	switch {
	case wantGit && len(s.worktrees) > 0:
		git.EXPECT().Checkout(s.cwd).Return(s.checkout.branch, s.checkout.root, s.checkout.mainRoot, s.checkout.project).Once()
	case wantGit:
		git.EXPECT().Branch(s.cwd).Return(s.checkout.branch).Once()
	}
	return newPath(t, s, git)
}

func newPath(t *testing.T, s setup, git *mocks.Git) *path.Path {
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
	return path.New(path.Deps{Session: sess, Config: cfg, Paint: paint, Git: git, Home: "/home/me"})
}

func TestRender(t *testing.T) {
	t.Parallel()

	const (
		repo = "/home/me/repo"
		wt   = "/home/me/repo/.worktrees/wt"
	)
	wtCheckout := checkout{branch: "feat", root: wt, mainRoot: repo, project: "repo"}
	tests := []struct {
		name string
		s    setup
		want string
	}{
		{"home abbreviated", setup{cwd: "/home/me/x", branch: true}, "<p>~/x</>"},
		{"regular repository", setup{cwd: repo, branch: true, checkout: checkout{branch: "main"}}, "<p>~/repo (</><b>main</><p>)</>"},
		{"regular repository with emoji", setup{cwd: repo, branch: true, emoji: true, checkout: checkout{branch: "main"}}, "<p>~/repo (</><b>B main</><p>)</>"},
		{"branch can be turned off", setup{cwd: repo}, "<p>~/repo</>"},
		{"strip applies", setup{cwd: repo, strip: []string{"~"}, branch: true, checkout: checkout{branch: "main"}}, "<p>repo (</><b>main</><p>)</>"},
		{"directory without a branch", setup{cwd: "/srv/x", branch: true}, "<p>/srv/x</>"},
		{"no directory hides the segment", setup{branch: true}, ""},

		{"worktree", setup{cwd: wt, branch: true, worktrees: []string{".worktrees"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"worktree with emoji", setup{cwd: wt, branch: true, emoji: true, worktrees: []string{".worktrees"}, checkout: wtCheckout}, "<p>repo (</><w>W feat</><p>)</>"},
		{"worktree subdirectory keeps what follows the root", setup{cwd: wt + "/pkg/sub", branch: true, worktrees: []string{".worktrees"}, checkout: wtCheckout}, "<p>repo/pkg/sub (</><w>feat</><p>)</>"},
		{"strip is ignored for a worktree", setup{cwd: wt, branch: true, strip: []string{"~"}, worktrees: []string{".worktrees"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"worktree without a branch name", setup{cwd: wt, branch: true, worktrees: []string{".worktrees"}, checkout: checkout{branch: "", root: wt, mainRoot: repo, project: "repo"}}, "<p>repo</>"},
		{"branch off shows the project alone", setup{cwd: wt, emoji: true, worktrees: []string{".worktrees"}, checkout: wtCheckout}, "<p>repo</>"},
		{"absolute entry without a main root", setup{cwd: wt, branch: true, worktrees: []string{"/home/me/repo/.worktrees"}, checkout: checkout{branch: "feat", root: wt, mainRoot: "", project: "repo"}}, "<p>repo (</><w>feat</><p>)</>"},
		{"relative entry without a main root", setup{cwd: wt, branch: true, worktrees: []string{"home"}, checkout: checkout{branch: "feat", root: wt, mainRoot: "", project: "repo"}}, "<p>~/repo/.worktrees/wt (</><b>feat</><p>)</>"},
		{"tilde entry", setup{cwd: wt, branch: true, worktrees: []string{"~/repo/.worktrees"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"tilde user entry is relative", setup{cwd: wt, branch: true, worktrees: []string{"~me/repo/.worktrees"}, checkout: wtCheckout}, "<p>~/repo/.worktrees/wt (</><b>feat</><p>)</>"},
		{"entry that cleans to the main root", setup{cwd: wt, branch: true, worktrees: []string{"./"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"entry with a trailing slash", setup{cwd: wt, branch: true, worktrees: []string{".worktrees/"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"entry with dot segments", setup{cwd: wt, branch: true, worktrees: []string{"a/../.worktrees"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"root entry is ignored", setup{cwd: wt, branch: true, worktrees: []string{"/"}, checkout: wtCheckout}, "<p>~/repo/.worktrees/wt (</><b>feat</><p>)</>"},
		{"any listed entry may match", setup{cwd: wt, branch: true, worktrees: []string{".claude/worktrees", ".worktrees"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"worktrees listed with branch off in a regular repository", setup{cwd: repo, worktrees: []string{".worktrees"}, checkout: checkout{branch: "main"}}, "<p>~/repo</>"},
		{"working directory with a trailing slash", setup{cwd: wt + "/", branch: true, worktrees: []string{".worktrees"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"working directory with a dot segment", setup{cwd: "/home/me/repo/.worktrees/./wt", branch: true, worktrees: []string{".worktrees"}, checkout: wtCheckout}, "<p>repo (</><w>feat</><p>)</>"},
		{"main repository itself is regular", setup{cwd: repo, branch: true, worktrees: []string{".worktrees"}, checkout: checkout{branch: "main"}}, "<p>~/repo (</><b>main</><p>)</>"},
		{"root equal to the entry is regular", setup{cwd: "/home/me/repo/.worktrees", branch: true, worktrees: []string{".worktrees"}, checkout: checkout{branch: "feat", root: "/home/me/repo/.worktrees", mainRoot: repo, project: "repo"}}, "<p>~/repo/.worktrees (</><b>feat</><p>)</>"},
		{"sibling named with the entry as a prefix is regular", setup{cwd: repo + "/.worktrees-old/wt", branch: true, worktrees: []string{".worktrees"}, checkout: checkout{branch: "feat", root: repo + "/.worktrees-old/wt", mainRoot: repo, project: "repo"}}, "<p>~/repo/.worktrees-old/wt (</><b>feat</><p>)</>"},
		{"directory outside the reported root is regular", setup{cwd: wt + "x", branch: true, worktrees: []string{".worktrees"}, checkout: wtCheckout}, "<p>~/repo/.worktrees/wtx (</><b>feat</><p>)</>"},
		{"worktree without a project is regular", setup{cwd: wt, branch: true, worktrees: []string{".worktrees"}, checkout: checkout{branch: "feat", root: wt, mainRoot: repo, project: ""}}, "<p>~/repo/.worktrees/wt (</><b>feat</><p>)</>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, build(t, tt.s).Render())
		})
	}
}

// The mocks fail on any call build did not expect, so each case also proves the other method is never called.
func TestGitReadsOnlyWhatIsNeeded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		s       setup
		wantGit bool
		want    string
	}{
		{"without worktrees only the branch is read", setup{cwd: "/home/me/repo", branch: true, checkout: checkout{branch: "main"}}, true, "<p>~/repo (</><b>main</><p>)</>"},
		{"with worktrees only the checkout is read", setup{cwd: "/home/me/repo", branch: true, worktrees: []string{".worktrees"}, checkout: checkout{branch: "main"}}, true, "<p>~/repo (</><b>main</><p>)</>"},
		{"with neither nothing is read", setup{cwd: "/home/me/repo", checkout: checkout{branch: "main"}}, false, "<p>~/repo</>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, buildStrict(t, tt.s, tt.wantGit).Render())
		})
	}
}

func TestNameAndKeys(t *testing.T) {
	t.Parallel()

	seg := build(t, setup{branch: true})
	assert.Equal(t, "path", seg.Name())
	assert.Equal(t, []string{"strip", "branch", "worktrees", "emoji"}, seg.Keys())
}

func TestHelp(t *testing.T) {
	t.Parallel()

	assert.Contains(t, build(t, setup{branch: true}).Help(), "branch")
	assert.NotContains(t, build(t, setup{}).Help(), "branch")
	assert.Contains(t, build(t, setup{strip: []string{"~/code", "/my srv"}}).Help(), "~/code, /my srv")
	assert.Contains(t, build(t, setup{worktrees: []string{".worktrees", "~/wt"}}).Help(), "under .worktrees, ~/wt")
	assert.NotContains(t, build(t, setup{}).Help(), "worktree")
}

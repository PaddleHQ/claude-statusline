package gitbranch_test

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/gitbranch"
	"github.com/PaddleHQ/claude-statusline/internal/payload"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sha = "0123456789abcdef0123456789abcdef01234567"

func write(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestRead(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files map[string]string
		dir   string
		want  string
	}{
		{"branch from a .git directory", map[string]string{"repo/.git/HEAD": "ref: refs/heads/main\n"}, "repo", "main"},
		{"branch with slashes", map[string]string{"repo/.git/HEAD": "ref: refs/heads/feature/x\n"}, "repo", "feature/x"},
		{"walks up from a subdirectory", map[string]string{"repo/.git/HEAD": "ref: refs/heads/main\n", "repo/a/b/file": ""}, "repo/a/b", "main"},
		{"detached HEAD shows seven characters", map[string]string{"repo/.git/HEAD": sha + "\n"}, "repo", "0123456"},
		{"worktree file with a relative gitdir", map[string]string{"wt/.git": "gitdir: ../main/.git/worktrees/wt\n", "main/.git/worktrees/wt/HEAD": "ref: refs/heads/topic\n"}, "wt", "topic"},
		{"worktree file with an absolute gitdir", map[string]string{"wt/.git": "gitdir: @TMP@/store/wt\n", "store/wt/HEAD": "ref: refs/heads/abs\n"}, "wt", "abs"},
		{"worktree detached", map[string]string{"wt/.git": "gitdir: ../store\n", "store/HEAD": sha}, "wt", "0123456"},
		{"no repository", map[string]string{"plain/file": ""}, "plain", ""},
		{"missing HEAD", map[string]string{"repo/.git/config": ""}, "repo", ""},
		{"unreadable gitdir target", map[string]string{"wt/.git": "gitdir: ../nowhere\n"}, "wt", ""},
		{"malformed .git file", map[string]string{"wt/.git": "nonsense\n"}, "wt", ""},
		{"empty gitdir", map[string]string{"wt/.git": "gitdir:\n"}, "wt", ""},
		{"garbage HEAD", map[string]string{"repo/.git/HEAD": "hello world\n"}, "repo", ""},
		{"short hex HEAD", map[string]string{"repo/.git/HEAD": "abc\n"}, "repo", ""},
		{"non-branch ref", map[string]string{"repo/.git/HEAD": "ref: refs/remotes/origin/x\n"}, "repo", ""},
		{"nearest repository wins", map[string]string{"outer/.git/HEAD": "ref: refs/heads/outer\n", "outer/inner/.git/HEAD": "ref: refs/heads/inner\n"}, "outer/inner", "inner"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmp := t.TempDir()
			for path, content := range tt.files {
				write(t, filepath.Join(tmp, path), strings.ReplaceAll(content, "@TMP@", tmp))
			}
			assert.Equal(t, tt.want, gitbranch.Read(filepath.Join(tmp, tt.dir)))
		})
	}
}

func TestReadRejectsRelativeAndMissingDirectories(t *testing.T) {
	t.Parallel()

	assert.Empty(t, gitbranch.Read(""))
	assert.Empty(t, gitbranch.Read("relative/dir"))
	assert.Empty(t, gitbranch.Read(filepath.Join(t.TempDir(), "does", "not", "exist")))
}

func TestReaderStripsControlCharacters(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write(t, filepath.Join(dir, ".git", "HEAD"), "ref: refs/heads/a\x1b[31mb\n")
	assert.Equal(t, "a[31mb", (&gitbranch.Reader{}).Branch(dir))
}

func TestReadDoesNotBlockOnFIFO(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	store := filepath.Join(tmp, "store")
	require.NoError(t, os.Mkdir(store, 0o750))
	require.NoError(t, syscall.Mkfifo(filepath.Join(store, "HEAD"), 0o600))
	repo := filepath.Join(tmp, "repo")
	require.NoError(t, os.Mkdir(repo, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: "+store+"\n"), 0o600))

	got := make(chan string, 1)
	go func() { got <- gitbranch.Read(repo) }()

	select {
	case branch := <-got:
		assert.Empty(t, branch)
	case <-time.After(2 * time.Second):
		require.FailNow(t, "Read blocked on a FIFO")
	}
}

func TestLinked(t *testing.T) {
	t.Parallel()

	const wtGit = "repo/.git/worktrees/wt/"
	tests := []struct {
		name     string
		files    map[string]string
		dir      string
		wantRoot string
		wantMain string
		wantOK   bool
	}{
		{"absolute gitdir and relative commondir", map[string]string{"repo/.worktrees/wt/.git": "gitdir: @TMP@/repo/.git/worktrees/wt\n", wtGit + "commondir": "../..\n"}, "repo/.worktrees/wt", "repo/.worktrees/wt", "repo", true},
		{"walks up from inside the worktree", map[string]string{"repo/.worktrees/wt/.git": "gitdir: @TMP@/repo/.git/worktrees/wt\n", wtGit + "commondir": "../..\n", "repo/.worktrees/wt/pkg/sub/file": ""}, "repo/.worktrees/wt/pkg/sub", "repo/.worktrees/wt", "repo", true},
		{"absolute commondir", map[string]string{"wt/.git": "gitdir: @TMP@/store/wt\n", "store/wt/commondir": "@TMP@/repo/.git\n", "repo/.git/HEAD": ""}, "wt", "wt", "repo", true},
		{"relative gitdir", map[string]string{"repo/.worktrees/wt/.git": "gitdir: ../../.git/worktrees/wt\n", wtGit + "commondir": "../..\n"}, "repo/.worktrees/wt", "repo/.worktrees/wt", "repo", true},
		{"commondir missing", map[string]string{"repo/.worktrees/wt/.git": "gitdir: @TMP@/repo/.git/worktrees/wt\n", wtGit + "HEAD": ""}, "repo/.worktrees/wt", "", "", false},
		{"empty commondir", map[string]string{"repo/.worktrees/wt/.git": "gitdir: @TMP@/repo/.git/worktrees/wt\n", wtGit + "commondir": "\n"}, "repo/.worktrees/wt", "", "", false},
		{"common directory not named .git", map[string]string{"wt/.git": "gitdir: @TMP@/x.git/worktrees/wt\n", "x.git/worktrees/wt/commondir": "../..\n"}, "wt", "", "", false},
		{"common directory is a file", map[string]string{"wt/.git": "gitdir: @TMP@/store/wt\n", "store/wt/commondir": "@TMP@/repo/.git\n", "repo/.git": "gitdir: elsewhere\n"}, "wt", "", "", false},
		{"common directory missing", map[string]string{"wt/.git": "gitdir: @TMP@/store/wt\n", "store/wt/commondir": "@TMP@/repo/.git\n"}, "wt", "", "", false},
		{"nested repository inside a worktree", map[string]string{"repo/.worktrees/wt/.git": "gitdir: @TMP@/repo/.git/worktrees/wt\n", wtGit + "commondir": "../..\n", "repo/.worktrees/wt/nested/.git/HEAD": ""}, "repo/.worktrees/wt/nested", "", "", false},
		{"main checkout", map[string]string{"repo/.git/HEAD": "ref: refs/heads/main\n"}, "repo", "", "", false},
		{"submodule without commondir", map[string]string{"repo/sub/.git": "gitdir: ../.git/modules/sub\n", "repo/.git/modules/sub/HEAD": ""}, "repo/sub", "", "", false},
		{"malformed .git file", map[string]string{"wt/.git": "nonsense\n"}, "wt", "", "", false},
		{"no repository", map[string]string{"plain/file": ""}, "plain", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmp := t.TempDir()
			for path, content := range tt.files {
				write(t, filepath.Join(tmp, path), strings.ReplaceAll(content, "@TMP@", tmp))
			}
			root, dotGit, info, ok := gitbranch.Locate(filepath.Join(tmp, tt.dir))
			var main string
			if ok {
				main, ok = gitbranch.Linked(info, gitbranch.GitDir(dotGit, info))
			}
			if !tt.wantOK {
				assert.False(t, ok)
				return
			}
			require.True(t, ok)
			assert.Equal(t, filepath.Join(tmp, tt.wantRoot), root)
			assert.Equal(t, filepath.Join(tmp, tt.wantMain), main)
		})
	}
}

func TestLocateRejectsRelativeAndEmptyDirectories(t *testing.T) {
	t.Parallel()

	_, _, _, ok := gitbranch.Locate("")
	assert.False(t, ok)
	_, _, _, ok = gitbranch.Locate("relative/dir")
	assert.False(t, ok)
}

func TestLinkedDoesNotBlockOnFIFO(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	store := filepath.Join(tmp, "store")
	require.NoError(t, os.Mkdir(store, 0o750))
	require.NoError(t, syscall.Mkfifo(filepath.Join(store, "commondir"), 0o600))
	repo := filepath.Join(tmp, "repo")
	require.NoError(t, os.Mkdir(repo, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: "+store+"\n"), 0o600))

	got := make(chan bool, 1)
	go func() {
		_, dotGit, info, ok := gitbranch.Locate(repo)
		if ok {
			_, ok = gitbranch.Linked(info, gitbranch.GitDir(dotGit, info))
		}
		got <- ok
	}()

	select {
	case ok := <-got:
		assert.False(t, ok)
	case <-time.After(2 * time.Second):
		require.FailNow(t, "locating a linked worktree blocked on a FIFO")
	}
}

// realTempDir resolves the platform's own symlinks (macOS /var) so each case controls every link.
func realTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	return dir
}

func mkdirs(t *testing.T, base string, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(base, d), 0o750))
	}
}

func symlink(t *testing.T, base, target, link string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(base, link)), 0o750))
	require.NoError(t, os.Symlink(filepath.Join(base, target), filepath.Join(base, link)))
}

func TestReconcile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, base string)
		w, m  string
		want  string
	}{
		{
			"worktree under the main root as spelled",
			func(t *testing.T, base string) { mkdirs(t, base, "repo/.worktrees/wt") },
			"repo/.worktrees/wt", "repo", "repo",
		},
		{
			"symlinked tmp",
			func(t *testing.T, base string) {
				mkdirs(t, base, "private/tmp/x/repo/.worktrees/wt")
				symlink(t, base, "private/tmp", "tmp")
			},
			"tmp/x/repo/.worktrees/wt", "private/tmp/x/repo", "tmp/x/repo",
		},
		{
			"symlinked home above both",
			func(t *testing.T, base string) {
				mkdirs(t, base, "Volumes/data/me/repo/.worktrees/wt", "home")
				symlink(t, base, "Volumes/data/me", "home/me")
			},
			"home/me/repo/.worktrees/wt", "Volumes/data/me/repo", "home/me/repo",
		},
		{
			"symlinked project directory",
			func(t *testing.T, base string) {
				mkdirs(t, base, "real/repo/.worktrees/wt")
				symlink(t, base, "real/repo", "proj")
			},
			"proj/.worktrees/wt", "real/repo", "proj",
		},
		{
			"symlinked worktrees directory",
			func(t *testing.T, base string) {
				mkdirs(t, base, "elsewhere/wt", "repo")
				symlink(t, base, "elsewhere", "repo/.worktrees")
			},
			"repo/.worktrees/wt", "repo", "repo",
		},
		{
			"worktree beside the main root",
			func(t *testing.T, base string) { mkdirs(t, base, "t/proj", "t/proj-wt/foo") },
			"t/proj-wt/foo", "t/proj", "t/proj",
		},
		{
			"worktree beside the main root under a symlinked parent",
			func(t *testing.T, base string) {
				mkdirs(t, base, "private/t/proj", "private/t/proj-wt/foo")
				symlink(t, base, "private/t", "t")
			},
			"t/proj-wt/foo", "private/t/proj", "t/proj",
		},
		{
			"worktree in an unrelated directory under a shared ancestor",
			func(t *testing.T, base string) { mkdirs(t, base, "a/wt", "b/repo") },
			"a/wt", "b/repo", "b/repo",
		},
		{
			"candidate that does not resolve to the main root",
			func(t *testing.T, base string) {
				mkdirs(t, base, "M/.worktrees/wt", "L")
				symlink(t, base, "M/.worktrees", "L/.worktrees")
			},
			"L/.worktrees/wt", "M", "",
		},
		{
			"worktree root equal to the main root",
			func(t *testing.T, base string) { mkdirs(t, base, "repo") },
			"repo", "repo", "repo",
		},
		{
			"symlinked worktree directory hides the remainder",
			func(t *testing.T, base string) {
				mkdirs(t, base, "real/repo/.worktrees/wt")
				symlink(t, base, "real/repo/.worktrees/wt", "wtlink")
			},
			"wtlink", "real/repo", "",
		},
		{
			"worktree beside the main root with a shared name prefix",
			func(t *testing.T, base string) { mkdirs(t, base, "repo-wt", "repo") },
			"repo-wt", "repo", "repo",
		},
		{
			"missing worktree root",
			func(t *testing.T, base string) { mkdirs(t, base, "b/repo") },
			"a/wt", "b/repo", "",
		},
		{
			"missing main root",
			func(t *testing.T, base string) { mkdirs(t, base, "a/wt") },
			"a/wt", "b/repo", "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			base := realTempDir(t)
			tt.setup(t, base)

			want := ""
			if tt.want != "" {
				want = filepath.Join(base, tt.want)
			}
			assert.Equal(t, want, gitbranch.Reconcile(filepath.Join(base, tt.w), filepath.Join(base, tt.m)))
		})
	}
}

func TestReconcileNeverUsesTheFilesystemRoot(t *testing.T) {
	t.Parallel()

	base := realTempDir(t)
	mkdirs(t, base, "wt")
	assert.Empty(t, gitbranch.Reconcile(filepath.Join(base, "wt"), "/"))
}

func worktreeFiles(main, wt string, extra map[string]string) map[string]string {
	files := map[string]string{
		main + "/.git/HEAD":                   "ref: refs/heads/main\n",
		wt + "/.git":                          "gitdir: @TMP@/" + main + "/.git/worktrees/wt\n",
		main + "/.git/worktrees/wt/commondir": "../..\n",
		main + "/.git/worktrees/wt/HEAD":      "ref: refs/heads/topic\n",
	}
	maps.Copy(files, extra)
	return files
}

func TestCheckout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		files       map[string]string
		links       map[string]string
		dir         string
		wantBranch  string
		wantRoot    string
		wantMain    string
		wantProject string
	}{
		{
			name:       "main repository",
			files:      worktreeFiles("repo", "repo/.worktrees/wt", nil),
			dir:        "repo",
			wantBranch: "main",
		},
		{
			name:        "linked worktree",
			files:       worktreeFiles("repo", "repo/.worktrees/wt", nil),
			dir:         "repo/.worktrees/wt",
			wantBranch:  "topic",
			wantRoot:    "repo/.worktrees/wt",
			wantMain:    "repo",
			wantProject: "repo",
		},
		{
			name:        "detached HEAD",
			files:       worktreeFiles("repo", "repo/.worktrees/wt", map[string]string{"repo/.git/worktrees/wt/HEAD": sha + "\n"}),
			dir:         "repo/.worktrees/wt",
			wantBranch:  "0123456",
			wantRoot:    "repo/.worktrees/wt",
			wantMain:    "repo",
			wantProject: "repo",
		},
		{
			name:        "subdirectory of the worktree",
			files:       worktreeFiles("repo", "repo/.worktrees/wt", map[string]string{"repo/.worktrees/wt/pkg/sub/file": ""}),
			dir:         "repo/.worktrees/wt/pkg/sub",
			wantBranch:  "topic",
			wantRoot:    "repo/.worktrees/wt",
			wantMain:    "repo",
			wantProject: "repo",
		},
		{
			name:        "reached through a symlink above the main root",
			files:       worktreeFiles("real/repo", "real/repo/.worktrees/wt", nil),
			links:       map[string]string{"link": "real"},
			dir:         "link/repo/.worktrees/wt",
			wantBranch:  "topic",
			wantRoot:    "link/repo/.worktrees/wt",
			wantMain:    "link/repo",
			wantProject: "repo",
		},
		{
			name:        "worktree in a directory beside the main repository",
			files:       worktreeFiles("repo", "wt", nil),
			dir:         "wt",
			wantBranch:  "topic",
			wantRoot:    "wt",
			wantMain:    "repo",
			wantProject: "repo",
		},
		{
			name:        "worktree beside the main repository",
			files:       worktreeFiles("proj", "proj-wt/foo", map[string]string{"proj-wt/foo/pkg/file": ""}),
			dir:         "proj-wt/foo/pkg",
			wantBranch:  "topic",
			wantRoot:    "proj-wt/foo",
			wantMain:    "proj",
			wantProject: "proj",
		},
		{
			name:  "no repository above the directory",
			files: map[string]string{"plain/file": ""},
			dir:   "plain",
		},
		{
			name: "directory that does not exist",
			dir:  "missing/dir",
		},
		{
			name:        "spellings that cannot be reconciled",
			files:       worktreeFiles("repo", "repo/.worktrees/wt", nil),
			links:       map[string]string{"wtlink": "repo/.worktrees/wt"},
			dir:         "wtlink",
			wantBranch:  "topic",
			wantRoot:    "wtlink",
			wantProject: "repo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmp := realTempDir(t)
			for path, content := range tt.files {
				write(t, filepath.Join(tmp, path), strings.ReplaceAll(content, "@TMP@", tmp))
			}
			for link, target := range tt.links {
				symlink(t, tmp, target, link)
			}
			join := func(rel string) string {
				if rel == "" {
					return ""
				}
				return filepath.Join(tmp, rel)
			}

			branch, root, main, project := (&gitbranch.Reader{}).Checkout(filepath.Join(tmp, tt.dir))
			assert.Equal(t, tt.wantBranch, branch)
			assert.Equal(t, join(tt.wantRoot), root)
			assert.Equal(t, join(tt.wantMain), main)
			assert.Equal(t, tt.wantProject, project)
		})
	}
}

func TestCheckoutStripsControlCharacters(t *testing.T) {
	t.Parallel()

	tmp := realTempDir(t)
	project := "a\x1b[31mb"
	files := worktreeFiles(project, project+"/.worktrees/wt", map[string]string{project + "/.git/worktrees/wt/HEAD": "ref: refs/heads/x\x1b[2Jy\n"})
	for path, content := range files {
		write(t, filepath.Join(tmp, path), strings.ReplaceAll(content, "@TMP@", tmp))
	}

	branch, root, main, name := (&gitbranch.Reader{}).Checkout(filepath.Join(tmp, project, ".worktrees", "wt"))
	for _, got := range []string{branch, root, main, name} {
		assert.NotEmpty(t, got)
		assert.NotContains(t, got, "\x1b")
	}
	assert.Equal(t, "x[2Jy", branch)
	assert.Equal(t, "a[31mb", name)
	assert.Equal(t, payload.Sanitize(filepath.Join(tmp, project)), main)
}

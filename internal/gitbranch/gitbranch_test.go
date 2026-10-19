package gitbranch_test

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/PaddleHQ/claude-statusline/internal/gitbranch"

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
			root, main, ok := gitbranch.Linked(filepath.Join(tmp, tt.dir))
			if !tt.wantOK {
				assert.False(t, ok)
				return
			}
			assert.True(t, ok)
			assert.Equal(t, filepath.Join(tmp, tt.wantRoot), root)
			assert.Equal(t, filepath.Join(tmp, tt.wantMain), main)
		})
	}
}

func TestLinkedRejectsRelativeAndEmptyDirectories(t *testing.T) {
	t.Parallel()

	_, _, ok := gitbranch.Linked("")
	assert.False(t, ok)
	_, _, ok = gitbranch.Linked("relative/dir")
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
		_, _, ok := gitbranch.Linked(repo)
		got <- ok
	}()

	select {
	case ok := <-got:
		assert.False(t, ok)
	case <-time.After(2 * time.Second):
		require.FailNow(t, "Linked blocked on a FIFO")
	}
}

package gitbranch_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

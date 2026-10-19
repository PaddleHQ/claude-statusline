// Package gitbranch finds the current branch by reading .git files, without running git.
package gitbranch

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/payload"
)

const (
	headPrefix = "ref: refs/heads/"
	gitdirTag  = "gitdir:"
	shortSHA   = 7
	maxRead    = 4096
)

// Reader finds branches by reading .git files and returns them safe to print.
type Reader struct{}

// Branch returns the branch checked out in dir, as Read does, with control characters removed.
func (g *Reader) Branch(dir string) string { return payload.Sanitize(Read(dir)) }

// Checkout returns the branch in dir and, when dir is inside a linked worktree, the
// worktree's root, the main repository's root and the project name. Both roots are in
// dir's spelling; mainRoot is "" when git's spelling cannot be matched to it, and the
// project name then comes from git's spelling. Every result is safe to print.
func (g *Reader) Checkout(dir string) (branch, worktreeRoot, mainRoot, project string) {
	root, dotGit, info, ok := locate(dir)
	if !ok {
		return "", "", "", ""
	}
	target := gitDir(dotGit, info)
	branch = payload.Sanitize(head(target))
	main, ok := linked(info, target)
	if !ok {
		return branch, "", "", ""
	}
	mainRoot = reconcile(root, main)
	project = filepath.Base(main)
	if mainRoot != "" {
		project = filepath.Base(mainRoot)
	}
	return branch, payload.Sanitize(root), payload.Sanitize(mainRoot), payload.Sanitize(project)
}

// Read returns the branch checked out in dir or any parent, the first seven characters
// of the commit when HEAD is detached, or "" when there is no readable repository.
func Read(dir string) string {
	_, dotGit, info, ok := locate(dir)
	if !ok {
		return ""
	}
	return head(gitDir(dotGit, info))
}

// locate walks up from the absolute dir to the nearest .git, returning the directory that holds it.
func locate(dir string) (root, dotGit string, info os.FileInfo, ok bool) {
	if !filepath.IsAbs(dir) {
		return "", "", nil, false
	}
	for dir = filepath.Clean(dir); ; dir = filepath.Dir(dir) {
		dotGit = filepath.Join(dir, ".git")
		if info, err := os.Stat(dotGit); err == nil {
			return dir, dotGit, info, true
		}
		if filepath.Dir(dir) == dir {
			return "", "", nil, false
		}
	}
}

// gitDir resolves a .git directory, or the directory a worktree or submodule's .git file points at.
func gitDir(dotGit string, info os.FileInfo) string {
	if info.IsDir() {
		return dotGit
	}
	line, ok := readFile(dotGit)
	if !ok {
		return ""
	}
	target, ok := strings.CutPrefix(strings.TrimSpace(line), gitdirTag)
	target = strings.TrimSpace(target)
	if !ok || target == "" {
		return ""
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(dotGit), target)
	}
	return target
}

func head(gitDir string) string {
	if gitDir == "" {
		return ""
	}
	content, ok := readFile(filepath.Join(gitDir, "HEAD"))
	if !ok {
		return ""
	}
	content = strings.TrimSpace(content)
	if branch, ok := strings.CutPrefix(content, headPrefix); ok {
		return branch
	}
	if isHex(content) && len(content) >= shortSHA {
		return content[:shortSHA]
	}
	return ""
}

func readFile(path string) (string, bool) {
	// A FIFO or device would block the open, and a status line must never hang.
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	f, err := os.Open(path) //nolint:gosec // G304: reading .git files under the working directory Claude Code passed is the feature
	if err != nil {
		return "", false
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxRead))
	if err != nil {
		return "", false
	}
	return string(data), true
}

func isHex(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return s != ""
}

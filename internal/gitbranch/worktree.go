package gitbranch

import (
	"os"
	"path/filepath"
	"strings"
)

// linked reports whether dir is inside a linked worktree, returning the worktree's root
// and the main repository's root. The common directory must be a directory named .git,
// which rejects worktrees of bare x.git repositories and submodules.
func linked(dir string) (root, main string, ok bool) {
	root, dotGit, info, ok := locate(dir)
	if !ok || info.IsDir() {
		return "", "", false
	}
	target := gitDir(dotGit, info)
	if target == "" {
		return "", "", false
	}
	content, ok := readFile(filepath.Join(target, "commondir"))
	if !ok {
		return "", "", false
	}
	common := strings.TrimSpace(content)
	if common == "" {
		return "", "", false
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(target, common)
	}
	common = filepath.Clean(common)
	if filepath.Base(common) != ".git" {
		return "", "", false
	}
	if commonInfo, err := os.Stat(common); err != nil || !commonInfo.IsDir() {
		return "", "", false
	}
	return root, filepath.Dir(common), true
}

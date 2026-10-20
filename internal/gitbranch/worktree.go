package gitbranch

import (
	"os"
	"path/filepath"
	"strings"
)

// linked reports whether the located .git, described by info and resolved to target, is a
// linked worktree's, returning the main repository's root. A plain submodule has no
// commondir; the common directory must also be a directory named .git, which rejects a
// worktree of a bare repository named x.git and a worktree of a submodule.
func linked(info os.FileInfo, target string) (main string, ok bool) {
	if info.IsDir() || target == "" {
		return "", false
	}
	content, ok := readFile(filepath.Join(target, "commondir"))
	if !ok {
		return "", false
	}
	common := strings.TrimSpace(content)
	if common == "" {
		return "", false
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(target, common)
	}
	common = filepath.Clean(common)
	if filepath.Base(common) != ".git" {
		return "", false
	}
	if commonInfo, err := os.Stat(common); err != nil || !commonInfo.IsDir() {
		return "", false
	}
	return filepath.Dir(common), true
}

// reconcile returns main, which is in git's real-path spelling, in the spelling of worktree
// (the working directory may come through a symlink such as macOS /tmp), or "" when they
// cannot be matched. The worktree may sit beside main, so the match goes through their deepest
// common ancestor, and the candidate must resolve back to main.
func reconcile(worktree, main string) string {
	if strings.HasPrefix(worktree, main+"/") {
		return main
	}
	realWorktree, err := filepath.EvalSymlinks(worktree)
	if err != nil {
		return ""
	}
	realMain, err := filepath.EvalSymlinks(main)
	if err != nil {
		return ""
	}
	for ancestor := realMain; ancestor != filepath.Dir(ancestor); ancestor = filepath.Dir(ancestor) {
		rest, ok := strings.CutPrefix(realWorktree, ancestor+"/")
		if !ok {
			continue
		}
		worktreeAncestor, ok := strings.CutSuffix(worktree, "/"+rest)
		if !ok {
			return ""
		}
		candidate := worktreeAncestor + strings.TrimPrefix(realMain, ancestor)
		if resolved, err := filepath.EvalSymlinks(candidate); err != nil || resolved != realMain {
			return ""
		}
		return candidate
	}
	return ""
}

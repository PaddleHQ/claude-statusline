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

// reconcile returns the main root m, which is in git's spelling, in the spelling of the
// worktree root w, or "" when the two cannot be matched. Git writes real paths, while the
// working directory may come through a symlink such as macOS /tmp.
func reconcile(w, m string) string {
	if strings.HasPrefix(w, m+"/") {
		return m
	}
	r, err := filepath.EvalSymlinks(w)
	if err != nil {
		return ""
	}
	mm, err := filepath.EvalSymlinks(m)
	if err != nil {
		return ""
	}
	rest, ok := strings.CutPrefix(r, mm+"/")
	if !ok || !strings.HasSuffix(w, "/"+rest) {
		return ""
	}
	return strings.TrimSuffix(w, "/"+rest)
}

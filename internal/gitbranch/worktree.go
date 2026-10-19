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

// reconcile returns the main root m, which is in git's spelling, in the spelling of the
// worktree root w, or "" when the two cannot be matched. Git writes real paths, while the
// working directory may come through a symlink such as macOS /tmp. The worktree may sit
// beside the main root rather than under it, so the match goes through their deepest
// common ancestor, and the candidate must resolve back to the main root.
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
	for a := mm; a != filepath.Dir(a); a = filepath.Dir(a) {
		rest, ok := strings.CutPrefix(r, a+"/")
		if !ok {
			continue
		}
		wa, ok := strings.CutSuffix(w, "/"+rest)
		if !ok {
			return ""
		}
		candidate := wa + strings.TrimPrefix(mm, a)
		if resolved, err := filepath.EvalSymlinks(candidate); err != nil || resolved != mm {
			return ""
		}
		return candidate
	}
	return ""
}

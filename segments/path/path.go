// Package path provides the path segment: the working directory, shortened, with the git branch, and a worktree as its main repository.
package path

import (
	_ "embed"
	"path/filepath"
	"strings"
)

// Defaults is the text of the segment's defaults.yaml, for layering under the user's config.
//
//go:embed defaults.yaml
var Defaults string

const name = "path"

//go:generate go run github.com/vektra/mockery/v2 --name=Session

// Session is what the segment needs to know about the session.
type Session interface {
	WorkingDir() string
}

//go:generate go run github.com/vektra/mockery/v2 --name=Config

// Config reads the segment's settings.
type Config interface {
	Strings(key string) []string
	Bool(key string) bool
}

//go:generate go run github.com/vektra/mockery/v2 --name=Painter

// Painter colors text.
type Painter interface {
	Path(s string) string
	Branch(s string) string
	Worktree(s string) string
	// Glyphs are the emoji for a regular branch and for a worktree branch.
	Glyphs() (branch, worktree string)
}

//go:generate go run github.com/vektra/mockery/v2 --name=Git

// Git reads the checkout a directory is in.
type Git interface {
	// Branch is the printable branch name, or "" when there is none.
	Branch(dir string) string
	// Checkout gives the printable branch name, and for a linked worktree its root, the main
	// repository's root and the main repository's name, all "" when unknown. The roots are in
	// dir's own spelling, and mainRoot is "" when that spelling cannot be reconciled.
	Checkout(dir string) (branch, worktreeRoot, mainRoot, project string)
}

// Deps is everything the segment is built from.
type Deps struct {
	Session Session
	Config  Config
	Paint   Painter
	Git     Git
	Home    string
}

// Path shows the working directory and its git branch.
type Path struct{ d Deps }

// New builds the path segment.
func New(d Deps) *Path { return &Path{d: d} }

// Name is the segment's name in the segments list.
func (p *Path) Name() string { return name }

// Help says what the segment shows, for the help command.
func (p *Path) Help() string {
	help := "The working directory, with your home directory shown as ~"
	if prefixes := p.d.Config.Strings("strip"); len(prefixes) > 0 {
		help += " and the prefixes " + strings.Join(prefixes, ", ") + " removed"
	}
	if p.d.Config.Bool("branch") {
		help += ", followed by the git branch in brackets"
	}
	help += "."
	if dirs := p.d.Config.Strings("worktrees"); len(dirs) > 0 {
		help += " A git worktree under " + strings.Join(dirs, ", ") + " is shown as the main repository's name."
	}
	return help
}

// Keys lists the config keys the segment reads, relative to its section.
func (p *Path) Keys() []string { return []string{"strip", "branch", "worktrees", "emoji"} }

// Render returns the segment's text, or "" to hide it.
func (p *Path) Render() string {
	dir := p.d.Session.WorkingDir()
	if dir == "" {
		return ""
	}
	showBranch := p.d.Config.Bool("branch")
	worktrees := p.d.Config.Strings("worktrees")
	strip := p.d.Config.Strings("strip")
	if !showBranch && len(worktrees) == 0 {
		return p.d.Paint.Path(displayPath(dir, p.d.Home, strip))
	}

	var branch, shown string
	var isWorktree bool
	if len(worktrees) == 0 {
		branch = p.d.Git.Branch(dir)
	} else {
		var root, mainRoot, project string
		branch, root, mainRoot, project = p.d.Git.Checkout(dir)
		shown, isWorktree = p.worktreePath(dir, root, mainRoot, project, worktrees)
	}
	if !isWorktree {
		shown = displayPath(dir, p.d.Home, strip)
	}
	if !showBranch || branch == "" {
		return p.d.Paint.Path(shown)
	}
	return p.d.Paint.Path(shown+" (") + p.paintBranch(branch, isWorktree) + p.d.Paint.Path(")")
}

// worktreePath is dir with everything up to the worktree root collapsed to the project name.
func (p *Path) worktreePath(dir, root, mainRoot, project string, entries []string) (string, bool) {
	if project == "" || !underWorktrees(root, mainRoot, p.d.Home, entries) {
		return "", false
	}
	rest, ok := strings.CutPrefix(filepath.Clean(dir), root)
	if !ok || (rest != "" && rest[0] != '/') {
		return "", false
	}
	return project + rest, true
}

func (p *Path) paintBranch(branch string, isWorktree bool) string {
	paint := p.d.Paint.Branch
	branchGlyph, worktreeGlyph := p.d.Paint.Glyphs()
	glyph := branchGlyph
	if isWorktree {
		paint, glyph = p.d.Paint.Worktree, worktreeGlyph
	}
	if p.d.Config.Bool("emoji") {
		branch = glyph + " " + branch
	}
	return paint(branch)
}

// underWorktrees reports whether root sits strictly under one of the entries. A relative entry
// is taken from the main repository and cannot match when that is unknown.
func underWorktrees(root, mainRoot, home string, entries []string) bool {
	if root == "" {
		return false
	}
	for _, entry := range entries {
		entry, ok := expandHome(entry, strings.TrimRight(home, "/"))
		if !ok {
			continue
		}
		if !filepath.IsAbs(entry) {
			if mainRoot == "" {
				continue
			}
			entry = mainRoot + "/" + entry
		}
		if entry = filepath.Clean(entry); entry != "/" && strings.HasPrefix(root, entry+"/") {
			return true
		}
	}
	return false
}

// displayPath strips the longest matching prefix, else abbreviates home to ~, else returns dir.
func displayPath(dir, home string, prefixes []string) string {
	home = strings.TrimRight(home, "/")
	if best := longestPrefix(dir, home, prefixes); best != "" {
		return dir[len(best)+1:]
	}
	if home != "" {
		if dir == home {
			return "~"
		}
		if rest, ok := strings.CutPrefix(dir, home+"/"); ok {
			return "~/" + rest
		}
	}
	return dir
}

func longestPrefix(dir, home string, prefixes []string) string {
	var best string
	for _, prefix := range prefixes {
		prefix, ok := expandHome(prefix, home)
		if !ok {
			continue
		}
		prefix = strings.TrimRight(prefix, "/")
		if prefix != "" && len(prefix) > len(best) && strings.HasPrefix(dir, prefix+"/") {
			best = prefix
		}
	}
	return best
}

// expandHome turns a leading ~ or ~/ into home. It reports false for such an entry when home is unknown.
func expandHome(entry, home string) (string, bool) {
	if rest, ok := strings.CutPrefix(entry, "~"); ok && (rest == "" || rest[0] == '/') {
		return home + rest, home != ""
	}
	return entry, true
}

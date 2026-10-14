// Package path provides the path segment: the working directory, shortened, with the git branch when there is one.
package path

import "strings"

const name = "path"

//go:generate go run github.com/vektra/mockery/v2 --name=Session

// Session is what the segment needs to know about the session.
type Session interface {
	Cwd() string
}

//go:generate go run github.com/vektra/mockery/v2 --name=Config

// Config reads the segment's settings.
type Config interface {
	String(key string) string
	Bool(key string) bool
}

//go:generate go run github.com/vektra/mockery/v2 --name=Painter

// Painter colors text.
type Painter interface {
	Path(s string) string
}

//go:generate go run github.com/vektra/mockery/v2 --name=Git

// Git finds the branch of a directory.
type Git interface {
	// Branch is the printable branch name, or "" when there is none.
	Branch(dir string) string
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
	if prefixes := strings.Fields(p.d.Config.String("strip")); len(prefixes) > 0 {
		help += " and the prefixes " + strings.Join(prefixes, " ") + " removed"
	}
	if p.d.Config.Bool("branch") {
		help += ", followed by the git branch in brackets"
	}
	return help + "."
}

// Keys lists the config keys the segment reads, relative to its section.
func (p *Path) Keys() []string { return []string{"strip", "branch"} }

// Render returns the segment's text, or "" to hide it.
func (p *Path) Render() string {
	dir := p.d.Session.Cwd()
	if dir == "" {
		return ""
	}
	out := displayPath(dir, p.d.Home, strings.Fields(p.d.Config.String("strip")))
	if p.d.Config.Bool("branch") {
		if branch := p.d.Git.Branch(dir); branch != "" {
			out += " (" + branch + ")"
		}
	}
	return p.d.Paint.Path(out)
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
		if rest, ok := strings.CutPrefix(prefix, "~"); ok && (rest == "" || rest[0] == '/') {
			if home == "" {
				continue
			}
			prefix = home + rest
		}
		prefix = strings.TrimRight(prefix, "/")
		if prefix != "" && len(prefix) > len(best) && strings.HasPrefix(dir, prefix+"/") {
			best = prefix
		}
	}
	return best
}

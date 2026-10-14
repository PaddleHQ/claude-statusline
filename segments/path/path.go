// Package path provides the path segment: the working directory, shortened, with the git branch when there is one.
package path

import (
	_ "embed"
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/gitbranch"
	"github.com/PaddleHQ/claude-statusline/internal/layout"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
	"github.com/PaddleHQ/claude-statusline/internal/plugin"
)

const name = "path"

//go:embed defaults.conf
var defaults string

// Plugin is the path plugin.
func Plugin() *plugin.Plugin {
	return &plugin.Plugin{Defaults: defaults, Segments: []plugin.Entry{{Name: name, New: New}}}
}

// New builds the path segment.
func New(p *plugin.Providers) plugin.Segment { return &path{p: p} }

type path struct{ p *plugin.Providers }

func (s *path) Name() string { return name }

func (s *path) Group() layout.Group { return layout.Identity }

func (s *path) Help() string {
	help := "The working directory, with your home directory shown as ~"
	if prefixes := strings.Fields(s.p.Config.String("strip")); len(prefixes) > 0 {
		help += " and the prefixes " + strings.Join(prefixes, " ") + " removed"
	}
	if s.p.Config.Bool("branch") {
		help += ", followed by the git branch in brackets"
	}
	return help + "."
}

func (s *path) Keys() []string { return []string{"strip", "branch"} }

func (s *path) Render(sess *plugin.Session) string {
	dir := sess.Payload.Cwd
	if dir == "" {
		return ""
	}
	th := s.p.Theme
	out := displayPath(dir, s.p.Home, strings.Fields(s.p.Config.String("strip")))
	if s.p.Config.Bool("branch") {
		if branch := payload.Sanitize(gitbranch.Read(dir)); branch != "" {
			out += " (" + branch + ")"
		}
	}
	return th.Paint(th.Path, out)
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

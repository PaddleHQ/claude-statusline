package segment

import (
	"strings"

	"github.com/PaddleHQ/claude-statusline/internal/gitbranch"
	"github.com/PaddleHQ/claude-statusline/internal/payload"
)

// path shows the working directory, shortened, with the git branch when there is one.
type path struct{}

func (path) Name() string { return "path" }

func (path) Keys() []string { return []string{"path.strip", "path.branch"} }

func (path) Render(ctx *Context) string {
	dir := ctx.Payload.Cwd
	if dir == "" {
		return ""
	}
	out := displayPath(dir, ctx.Home, strings.Fields(ctx.Config.String("path.strip")))
	if ctx.Config.Bool("path.branch") {
		if branch := payload.Sanitize(gitbranch.Read(dir)); branch != "" {
			out += " (" + branch + ")"
		}
	}
	return ctx.Theme.Paint(ctx.Theme.Path, out)
}

// displayPath strips the longest matching prefix, else abbreviates home to ~, else returns dir.
func displayPath(dir, home string, prefixes []string) string {
	home = strings.TrimRight(home, "/")
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
	if best != "" {
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

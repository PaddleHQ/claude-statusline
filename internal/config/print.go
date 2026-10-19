package config

import "strings"

const printHeader = `# claude-statusline settings, all commented out so the defaults still apply.
# Save your file as config.yaml in the claude-statusline folder of your Claude config directory.
# "claude-statusline help" shows the exact path.
# To change a setting, remove the "# " in front of it.
# When it sits in a section, remove the "# " in front of the section header too.
# Delete the settings you do not change, so that future defaults reach you.
# If you toggle comments in an editor, the explanations stay commented,
# because only the lines you pick lose their "# ".
`

// CommentedDefaults returns the global and plugin defaults with every line commented out, built from the text so the order never varies.
func CommentedDefaults(pluginDefaults ...string) string {
	texts := append([]string{printHeader, Defaults}, pluginDefaults...)
	parts := make([]string, len(texts))
	for i, text := range texts {
		text = strings.ReplaceAll(text, "\r\n", "\n")
		parts[i] = strings.TrimRight(text, "\n")
	}
	lines := strings.Split(strings.Join(parts, "\n\n"), "\n")
	var b strings.Builder
	for _, line := range lines {
		if line == "" {
			b.WriteString("#\n")
			continue
		}
		b.WriteString("# " + line + "\n")
	}
	return b.String()
}

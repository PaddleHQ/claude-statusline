// Package rendertest loads render test cases: one txtar file per case, holding the input, the settings
// and the exact output expected. The status line's tests and the documentation image generator share it.
package rendertest

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/txtar"
)

const (
	// Extension is the file extension of a case.
	Extension = ".txtar"

	// TmpMarker stands for the case's temporary directory in the sections that name paths.
	TmpMarker = "@TMP@"

	privateDir     = 0o700
	privateFile    = 0o600
	executableFile = 0o700
)

// Section names.
const (
	Input         = "input.json"
	Config        = "config"
	Env           = "env"
	Cache         = "cache"
	ExpectedCache = "expected_cache"
	Tree          = "tree"
	Command       = "command"
	Output        = "output"
)

var (
	// ErrMissingSection is returned when a case lacks a section it requires.
	ErrMissingSection = errors.New("missing section")
	// ErrNoCases is returned when a directory holds no cases.
	ErrNoCases = errors.New("no cases found")
)

// Case is one parsed txtar file.
type Case struct {
	// Name is the file name without its extension.
	Name string

	path    string
	archive *txtar.Archive
}

// Load parses every case in dir, ordered by file name.
func Load(dir string) ([]*Case, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*"+Extension))
	if err != nil {
		return nil, fmt.Errorf("list cases: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("%w in %s", ErrNoCases, dir)
	}
	cases := make([]*Case, 0, len(paths))
	for _, path := range paths {
		archive, err := txtar.ParseFile(path)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		name := strings.TrimSuffix(filepath.Base(path), Extension)
		cases = append(cases, &Case{Name: name, path: path, archive: archive})
	}
	return cases, nil
}

// Description is the one-line comment before the first section.
func (c *Case) Description() string {
	return strings.TrimSpace(string(c.archive.Comment))
}

// Section returns a section as written, which always ends in a newline unless it is empty, and whether
// the case has it.
func (c *Case) Section(name string) (string, bool) {
	for _, f := range c.archive.Files {
		if f.Name == name {
			return string(f.Data), true
		}
	}
	return "", false
}

// Required returns a section the case must have.
func (c *Case) Required(name string) (string, error) {
	data, ok := c.Section(name)
	if !ok {
		return "", fmt.Errorf("%s: %w %q", c.Name, ErrMissingSection, name)
	}
	return data, nil
}

// Expected returns a section holding expected text without the one newline txtar adds to end it.
func (c *Case) Expected(name string) (string, bool) {
	data, ok := c.Section(name)
	return strings.TrimSuffix(data, "\n"), ok
}

// Env returns defaults overlaid with the KEY=value lines of the env section; lines starting with # are ignored.
func (c *Case) Env(defaults map[string]string) map[string]string {
	env := maps.Clone(defaults)
	data, _ := c.Section(Env)
	for line := range strings.SplitSeq(data, "\n") {
		if k, v, ok := strings.Cut(line, "="); ok && !strings.HasPrefix(k, "#") {
			env[k] = v
		}
	}
	return env
}

// WriteConfig writes the config section, with the temp marker replaced, as the user's config file under tmp.
func (c *Case) WriteConfig(tmp string) error {
	data, ok := c.Section(Config)
	if !ok {
		return nil
	}
	appDir := filepath.Join(tmp, "claude-statusline")
	if err := os.MkdirAll(appDir, privateDir); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	text := strings.ReplaceAll(data, TmpMarker, tmp)
	if err := os.WriteFile(filepath.Join(appDir, "config"), []byte(text), privateFile); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// WriteCache seeds the rate-limit cache from the cache section and returns the cache file's path.
func (c *Case) WriteCache(tmp string) (string, error) {
	file := filepath.Join(tmp, "claude-statusline", "cache", "ratelimits")
	data, ok := c.Section(Cache)
	if !ok {
		return file, nil
	}
	if err := os.MkdirAll(filepath.Dir(file), privateDir); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}
	if err := os.WriteFile(file, []byte(data), privateFile); err != nil {
		return "", fmt.Errorf("write cache: %w", err)
	}
	return file, nil
}

// WriteTree creates the files listed in the tree section under tmp: one path=content per line, a trailing / for
// a directory, a leading +x for an executable file, and \n in content for a newline.
func (c *Case) WriteTree(tmp string) error {
	data, _ := c.Section(Tree)
	for line := range strings.SplitSeq(data, "\n") {
		if line == "" {
			continue
		}
		if err := writeTreeEntry(tmp, line); err != nil {
			return err
		}
	}
	return nil
}

// Input returns the input.json section with the temp marker replaced.
func (c *Case) Input(tmp string) ([]byte, error) {
	data, err := c.Required(Input)
	if err != nil {
		return nil, err
	}
	return []byte(strings.ReplaceAll(data, TmpMarker, tmp)), nil
}

// Update sets each named section to text, adding the newline that ends a txtar section, and saves the file.
// A section whose text already matches is left as it is, so a case that still passes is never rewritten.
// Sections the case does not have are not added, and neither are the comment or other sections touched.
func (c *Case) Update(sections map[string]string) error {
	changed := false
	for i, f := range c.archive.Files {
		text, ok := sections[f.Name]
		if !ok || sameText(f.Name, string(f.Data), text) {
			continue
		}
		c.archive.Files[i].Data = []byte(sectionData(text))
		changed = true
	}
	if !changed {
		return nil
	}
	if err := os.WriteFile(c.path, txtar.Format(c.archive), privateFile); err != nil {
		return fmt.Errorf("write %s: %w", c.path, err)
	}
	return nil
}

// sameText compares a stored section with fresh text the way the tests do: the rate-limit cache ignores
// surrounding whitespace, and everything else must match exactly.
func sameText(name, stored, fresh string) bool {
	if name == ExpectedCache {
		return strings.TrimSpace(stored) == strings.TrimSpace(fresh)
	}
	return strings.TrimSuffix(stored, "\n") == fresh
}

func sectionData(text string) string {
	if text == "" {
		return ""
	}
	return text + "\n"
}

func writeTreeEntry(tmp, line string) error {
	mode := os.FileMode(privateFile)
	if rest, ok := strings.CutPrefix(line, "+x "); ok {
		line, mode = rest, executableFile
	}
	rel, content, _ := strings.Cut(line, "=")
	content = strings.ReplaceAll(content, `\n`, "\n")
	path := filepath.Join(tmp, rel)
	if strings.HasSuffix(rel, "/") {
		if err := os.MkdirAll(path, privateDir); err != nil {
			return fmt.Errorf("create %s: %w", rel, err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), privateDir); err != nil {
		return fmt.Errorf("create directory for %s: %w", rel, err)
	}
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(content, TmpMarker, tmp)+"\n"), mode); err != nil {
		return fmt.Errorf("write %s: %w", rel, err)
	}
	return nil
}

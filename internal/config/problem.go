package config

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/PaddleHQ/claude-statusline/internal/payload"

	"go.yaml.in/yaml/v3"
)

const (
	reasonNotMapping = "the top level must be a mapping"
	reasonNotRegular = "not a regular file"
	reasonTooLarge   = "file larger than 1 MiB"
	reportNote       = "The line is where the parser noticed the problem; the mistake may be just before it."
	contextBefore    = 3
	contextAfter     = 2
)

var yamlLine = regexp.MustCompile(`line (\d+): `)

// Problem says why the user's config file could not be used, so its settings were ignored.
type Problem struct {
	// Path is the config file, or empty when the text did not come from a file.
	Path string
	// Line is where the parser noticed the problem, counting from 1, or 0 when it gave none.
	Line int
	// Reason is the parser's or the operating system's message, not yet cleaned for printing.
	Reason string
	source string
}

// Problem returns why the user's config file could not be used, or nil when it was usable.
func (c *Config) Problem() *Problem {
	return c.problem
}

// UnknownKeys lists the keys in the user's file that no defaults file has, as "section.key", sorted.
func (c *Config) UnknownKeys(pluginDefaults ...string) []string {
	defaults := mergeDefaults(pluginDefaults)
	var unknown []string
	for key := range flatten(c.user) {
		if _, ok := defaults[key]; !ok {
			unknown = append(unknown, key)
		}
	}
	slices.Sort(unknown)
	return unknown
}

// Report describes the problem for a person: the line and the reason, then the file's lines around it.
func (p *Problem) Report() string {
	reason := Clean(p.Reason)
	if p.Line == 0 {
		return reason
	}
	head := strconv.Itoa(p.Line) + " " + reason
	lines := splitLines(p.source)
	if p.Line > len(lines)+1 {
		return head
	}
	return head + "\n\n" + p.context(lines) + "\n\n" + reportNote
}

func (p *Problem) context(lines []string) string {
	first := max(1, p.Line-contextBefore)
	last := max(p.Line, min(len(lines), p.Line+contextAfter))
	width := len(strconv.Itoa(last))
	rows := make([]string, 0, last-first+1)
	for n := first; n <= last; n++ {
		marker := " "
		if n == p.Line {
			marker = ">"
		}
		text := ""
		if n <= len(lines) {
			text = Clean(lines[n-1])
		}
		rows = append(rows, strings.TrimRight(fmt.Sprintf("%s %*d | %s", marker, width, n, text), " "))
	}
	return strings.Join(rows, "\n")
}

// Clean makes text safe to print: no control characters and no invisible format characters such as bidi overrides.
func Clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, payload.Sanitize(s))
}

// splitLines breaks text where the YAML parser counts a new line, so line numbers match.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch r {
		case '\r':
			lines = append(lines, s[start:i])
			if i+1 < len(s) && s[i+1] == '\n' {
				size = 2
			}
			start = i + size
		case '\n', '\u0085', '\u2028', '\u2029':
			lines = append(lines, s[start:i])
			start = i + size
		}
		i += size
	}
	return append(lines, s[start:])
}

// parseProblem never returns part of a broken file: the parser fills the mapping for some errors, such as a nested duplicate key.
func parseProblem(text string) (map[string]any, *Problem) {
	var root any
	if err := yaml.Unmarshal([]byte(text), &root); err != nil {
		return nil, yamlProblem(text, err)
	}
	if root == nil {
		return nil, nil
	}
	user := mapOf(root)
	if user == nil {
		return nil, &Problem{Reason: reasonNotMapping, source: text}
	}
	return user, nil
}

func yamlProblem(text string, err error) *Problem {
	msg := err.Error()
	p := &Problem{source: text}
	if loc := yamlLine.FindStringSubmatchIndex(msg); loc != nil {
		p.Line, _ = strconv.Atoi(msg[loc[2]:loc[3]])
		msg = msg[loc[1]:]
	} else {
		msg = strings.TrimPrefix(msg, "yaml: ")
	}
	p.Reason, _, _ = strings.Cut(msg, "\n")
	return p
}

// readProblem turns a failure to read the file into a problem; a missing file is not one.
func readProblem(path string, err error) *Problem {
	var reason string
	var pathErr *fs.PathError
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case errors.Is(err, errNotRegular):
		reason = reasonNotRegular
	case errors.Is(err, errTooLarge):
		reason = reasonTooLarge
	case errors.As(err, &pathErr):
		reason = pathErr.Err.Error()
	default:
		reason = err.Error()
	}
	return &Problem{Path: path, Reason: reason}
}

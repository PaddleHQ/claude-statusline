// Package config layers the user's settings over the defaults embedded in the binary.
package config

import (
	_ "embed"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

//go:embed defaults.conf
var defaultsText string

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)?$`)

// maxDigits keeps every accepted number comfortably inside an int.
const maxDigits = 12

// Config resolves each key from the user's file, falling back to the default.
// A key absent from defaults.conf is a programming error that a test rejects,
// so its getters return the zero value.
type Config struct {
	defaults map[string]string
	user     map[string]string
}

// New builds a Config from the text of a user's config file.
func New(userText string) *Config {
	return &Config{defaults: parse(defaultsText), user: parse(userText)}
}

// Load reads the user's config file; a missing or unreadable file means defaults only.
func Load(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return New("")
	}
	return New(string(data))
}

// Dir returns the Claude config directory, or "" when neither the environment nor HOME names one.
func Dir(getenv func(string) string) string {
	if dir := getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	if home := getenv("HOME"); home != "" {
		return filepath.Join(home, ".claude")
	}
	return ""
}

// AppDir returns the directory holding the user's config file, or "" when there is no config directory.
func AppDir(getenv func(string) string) string {
	dir := Dir(getenv)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "claude-statusline")
}

// Path returns where the user's config file lives, or "" when there is no config directory.
func Path(getenv func(string) string) string {
	dir := AppDir(getenv)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "config")
}

// DefaultKeys lists every key in defaults.conf.
func DefaultKeys() []string {
	defaults := parse(defaultsText)
	keys := make([]string, 0, len(defaults))
	for k := range defaults {
		keys = append(keys, k)
	}
	return keys
}

// String returns a non-empty value.
func (c *Config) String(key string) string {
	if v := c.user[key]; v != "" {
		return v
	}
	return c.defaults[key]
}

// Int returns a non-negative whole number.
func (c *Config) Int(key string) int {
	if n, ok := ParseNumber(c.user[key]); ok {
		return n
	}
	n, _ := ParseNumber(c.defaults[key])
	return n
}

// Bool returns true or false; anything else falls back.
func (c *Config) Bool(key string) bool {
	switch c.user[key] {
	case "true":
		return true
	case "false":
		return false
	}
	return c.defaults[key] == "true"
}

// ParseNumber accepts plain decimal digits only, so signs, exponents and absurd lengths fall back.
func ParseNumber(s string) (int, bool) {
	if s == "" || len(strings.TrimLeft(s, "0")) > maxDigits {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

// parse reads key = value lines as data; nothing in a value is ever interpreted.
func parse(text string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if !keyPattern.MatchString(key) {
			continue
		}
		values[key] = strings.TrimSpace(value)
	}
	return values
}

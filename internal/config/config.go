// Package config layers the user's settings over the defaults embedded in the binary.
package config

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"
)

// FileName is the user's config file inside the app directory.
const FileName = "config.yaml"

var (
	errTooLarge   = errors.New("config file too large")
	errNotRegular = errors.New("config file is not a regular file")
)

// Defaults is the text of the global defaults.yaml, which sits under every plugin's defaults.
//
//go:embed defaults.yaml
var Defaults string

// maxDigits keeps every accepted number comfortably inside an int.
const maxDigits = 12

// maxInt is the largest whole number a setting may hold.
const maxInt = 999_999_999_999

// maxFileBytes caps how much of the user's config file is read.
const maxFileBytes = 1 << 20

// Config resolves each key from the user's file, falling back to the default.
// A key absent from every defaults file is a programming error that a test rejects,
// so its getters return the zero value.
type Config struct {
	defaults map[string]any
	user     map[string]any
	problem  *Problem
}

// New builds a Config from the text of a user's config file. The defaults are the embedded global ones
// layered under the defaults files of the plugins, given as text. Text that is not a YAML mapping is ignored,
// and Problem says why.
func New(userText string, pluginDefaults ...string) *Config {
	user, problem := parseProblem(userText)
	return &Config{defaults: mergeDefaults(pluginDefaults), user: user, problem: problem}
}

// Load reads the user's config file; a missing or unusable file means defaults only, and Problem says why
// a file that exists could not be used.
func Load(path string, pluginDefaults ...string) *Config {
	data, err := readLimited(path)
	if err != nil {
		cfg := New("", pluginDefaults...)
		cfg.problem = readProblem(path, err)
		return cfg
	}
	cfg := New(string(data), pluginDefaults...)
	if cfg.problem != nil {
		cfg.problem.Path = path
	}
	return cfg
}

// readLimited refuses anything but a regular file before opening it, because opening a FIFO blocks.
func readLimited(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat config file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, errNotRegular
	}
	f, err := os.Open(path) //nolint:gosec // G304: reading the config file the user named is the feature
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	if len(data) > maxFileBytes {
		return nil, errTooLarge
	}
	return data, nil
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

// StateDirEnv names the variable the command line sets to the directory holding the binary's private state when
// it was installed by the plugin. The binary works that out for itself, so a value from the real environment is
// replaced before it gets here.
const StateDirEnv = "CLAUDE_STATUSLINE_STATE_DIR"

// StateDir returns the directory for caches and setup markers, or "" when there is none. It is the plugin's data
// directory when the binary lives in one, and the config file's directory otherwise.
func StateDir(getenv func(string) string) string {
	if dir := getenv(StateDirEnv); dir != "" {
		return dir
	}
	return AppDir(getenv)
}

// PluginDataDir returns the plugin data directory the executable was installed into, or "" when it lives anywhere
// else. Claude Code keeps a plugin's data at <config dir>/plugins/data/<plugin>, and the plugin installs the binary
// into its bin directory.
func PluginDataDir(exe string) string {
	bin := filepath.Dir(exe)
	data := filepath.Dir(bin)
	plugins := filepath.Dir(filepath.Dir(data))
	if filepath.Base(bin) == "bin" && filepath.Base(filepath.Dir(data)) == "data" && filepath.Base(plugins) == "plugins" {
		return data
	}
	return ""
}

// Path returns where the user's config file lives, or "" when there is no config directory.
func Path(getenv func(string) string) string {
	dir := AppDir(getenv)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, FileName)
}

// DefaultKeys lists every key in the global defaults.yaml and in the given plugin defaults, as "section.key" or "key".
func DefaultKeys(pluginDefaults ...string) []string {
	defaults := mergeDefaults(pluginDefaults)
	keys := make([]string, 0, len(defaults))
	for k := range defaults {
		keys = append(keys, k)
	}
	return keys
}

// UncommentedKeys lists the keys of the given defaults text that have no comment on the lines above them.
func UncommentedKeys(text string) []string {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(text), &root); err != nil || len(root.Content) == 0 {
		return nil
	}
	var missing []string
	collectUncommented(root.Content[0], "", &missing)
	return missing
}

func collectUncommented(mapping *yaml.Node, prefix string, missing *[]string) {
	if mapping.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key, value := mapping.Content[i], mapping.Content[i+1]
		name := prefix + key.Value
		if value.Kind == yaml.MappingNode {
			collectUncommented(value, name+".", missing)
			continue
		}
		if key.HeadComment == "" {
			*missing = append(*missing, name)
		}
	}
}

// Section returns the keys under name, so Section("model").Bool("effort") reads model.effort.
// An empty name reads keys as they are.
func (c *Config) Section(name string) *Section {
	return &Section{cfg: c, prefix: name}
}

// String returns a non-empty string without control characters.
func (c *Config) String(key string) string {
	if s, ok := stringOf(c.lookupUser(key)); ok {
		return s
	}
	s, _ := stringOf(c.defaults[key])
	return s
}

// Strings returns a non-empty list of strings without control characters; empty entries are dropped.
func (c *Config) Strings(key string) []string {
	if l, ok := stringsOf(c.lookupUser(key)); ok {
		return l
	}
	l, _ := stringsOf(c.defaults[key])
	return l
}

// StringsOrNone is Strings, except that an empty list in the user's config is kept as no strings rather than falling back,
// so a user can clear a list whose default is not empty.
func (c *Config) StringsOrNone(key string) []string {
	if items, ok := c.lookupUser(key).([]any); ok && len(items) == 0 {
		return nil
	}
	return c.Strings(key)
}

// HasUserList reports whether the user's config sets key to a list that Strings uses rather than the default.
func (c *Config) HasUserList(key string) bool {
	_, ok := stringsOf(c.lookupUser(key))
	return ok
}

// Int returns a whole number from 0 to 999999999999.
func (c *Config) Int(key string) int {
	if n, ok := intOf(c.lookupUser(key)); ok {
		return n
	}
	n, _ := intOf(c.defaults[key])
	return n
}

// Bool returns true or false; anything else falls back.
func (c *Config) Bool(key string) bool {
	if b, ok := c.lookupUser(key).(bool); ok {
		return b
	}
	b, _ := c.defaults[key].(bool)
	return b
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

func (c *Config) lookupUser(key string) any {
	section, name, nested := strings.Cut(key, ".")
	if !nested {
		return c.user[key]
	}
	return mapOf(c.user[section])[name]
}

func stringOf(v any) (string, bool) {
	s, ok := v.(string)
	if !ok || s == "" || strings.ContainsFunc(s, unicode.IsControl) {
		return "", false
	}
	return s, true
}

func stringsOf(v any) ([]string, bool) {
	items, ok := v.([]any)
	if !ok {
		return nil, false
	}
	list := make([]string, 0, len(items))
	for _, item := range items {
		s, isString := item.(string)
		if !isString || strings.ContainsFunc(s, unicode.IsControl) {
			return nil, false
		}
		if s != "" {
			list = append(list, s)
		}
	}
	return list, len(list) > 0
}

func intOf(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, n >= 0 && n <= maxInt
	case int64:
		return int(n), n >= 0 && n <= maxInt
	case uint64:
		return int(n), n <= maxInt //nolint:gosec // G115: n is checked against maxInt
	}
	return 0, false
}

// mapOf returns v as a string-keyed mapping, or nil.
func mapOf(v any) map[string]any {
	switch m := v.(type) {
	case map[string]any:
		return m
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			if name, ok := k.(string); ok {
				out[name] = val
			}
		}
		return out
	}
	return nil
}

// parse reads YAML as data; nothing in a value is ever interpreted. Anything that is not a YAML mapping gives nil.
func parse(text string) map[string]any {
	user, _ := parseProblem(text)
	return user
}

// flatten turns a mapping of sections into "section.key" entries; a top-level key holding a mapping is a section.
func flatten(root map[string]any) map[string]any {
	flat := map[string]any{}
	for k, v := range root {
		if section := mapOf(v); section != nil {
			for name, val := range section {
				flat[k+"."+name] = val
			}
			continue
		}
		flat[k] = v
	}
	return flat
}

func mergeDefaults(pluginDefaults []string) map[string]any {
	merged := flatten(parse(Defaults))
	for _, text := range pluginDefaults {
		maps.Copy(merged, flatten(parse(text)))
	}
	return merged
}

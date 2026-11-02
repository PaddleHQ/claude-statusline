package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

// PluginName is the name of the plugin whose userConfig values overlay the config file.
const PluginName = "statusline"

// SettingsFileName is the Claude Code user settings file that holds the saved plugin options.
const SettingsFileName = "settings.json"

// Source names where an effective value came from.
const (
	SourcePlugin   = "plugin options"
	SourceFile     = "config.yaml"
	SourceDefaults = "default"
)

// optionDefault is the tri-state value that leaves a setting to config.yaml.
const optionDefault = "default"

// OptionKind says how a plugin option maps onto a config value.
type OptionKind int

const (
	// KindString is a single string.
	KindString OptionKind = iota
	// KindList is a list of strings.
	KindList
	// KindChoice is a string where "default" means no opinion, such as a picker of built-in names.
	KindChoice
	// KindToggle is "default", "on" or "off", standing for no opinion, true and false.
	KindToggle
)

// Option maps a userConfig key declared in plugin.json onto a config key.
type Option struct {
	Name string
	Key  string
	Kind OptionKind
}

var options = []Option{
	{"theme", "theme", KindChoice},
	{"theme_file", "theme", KindString},
	{"segments", "segments", KindList},
	{"separator", "separator", KindString},
	{"path_strip", "path.strip", KindList},
	{"path_worktrees", "path.worktrees", KindList},
	{"path_branch", "path.branch", KindToggle},
	{"path_emoji", "path.emoji", KindToggle},
	{"model_effort", "model.effort", KindToggle},
	{"snapshot_enabled", "snapshot.enabled", KindToggle},
	{"snapshot_path", "snapshot.path", KindString},
}

// Options lists the plugin options that overlay config keys, in the order they are reported.
func Options() []Option {
	return slices.Clone(options)
}

// PluginOptions is what the user's settings.json holds for this plugin.
type PluginOptions struct {
	// Path is the settings file that was read.
	Path string
	// Entry is the pluginConfigs key the options came from, such as statusline@claude-statusline; empty when none applied.
	Entry string
	// Issues are problems reading the file or choosing an entry, for config-check; the status line never shows them.
	Issues []string
	values map[string]any
}

// Active says whether an enabled plugin entry supplied options.
func (o *PluginOptions) Active() bool {
	return o != nil && o.Entry != ""
}

// settingsFile is the part of settings.json this package reads.
type settingsFile struct {
	EnabledPlugins map[string]any `json:"enabledPlugins"`
	PluginConfigs  map[string]struct {
		Options map[string]any `json:"options"`
	} `json:"pluginConfigs"`
}

// LoadPluginOptions reads the saved plugin options from the settings file in dir. An absent, unreadable or malformed
// file gives no options, with the reason in Issues unless the file is simply missing.
func LoadPluginOptions(dir string) *PluginOptions {
	opts := &PluginOptions{}
	if dir == "" {
		return opts
	}
	opts.Path = filepath.Join(dir, SettingsFileName)
	data, err := readLimited(opts.Path)
	if err != nil {
		if p := readProblem(opts.Path, err); p != nil {
			opts.Issues = append(opts.Issues, p.Reason)
		}
		return opts
	}
	var settings settingsFile
	if err := json.Unmarshal(data, &settings); err != nil {
		opts.Issues = append(opts.Issues, fmt.Sprintf("not usable as settings: %v", err))
		return opts
	}
	opts.choose(&settings)
	return opts
}

// choose picks the enabled entry for this plugin, the first by sorted key when there are several.
func (o *PluginOptions) choose(settings *settingsFile) {
	var entries []string
	for key := range settings.PluginConfigs {
		name, _, _ := strings.Cut(key, "@")
		if name == PluginName && settings.EnabledPlugins[key] == true {
			entries = append(entries, key)
		}
	}
	if len(entries) == 0 {
		return
	}
	slices.Sort(entries)
	o.Entry = entries[0]
	o.values = settings.PluginConfigs[o.Entry].Options
	if len(entries) > 1 {
		o.Issues = append(o.Issues, fmt.Sprintf("several enabled entries hold options for %s (%s); using %s",
			PluginName, strings.Join(entries, ", "), o.Entry))
	}
}

// set says whether the user chose a usable value for the option: present, and neither empty nor "default".
func (o Option) set(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case string:
		switch o.Kind {
		case KindList:
			return len(splitList(x)) > 0
		case KindToggle, KindChoice:
			return x != "" && x != optionDefault
		}
		return x != ""
	case []any:
		if o.Kind == KindList {
			_, ok := stringsOf(x)
			return ok
		}
		return len(x) > 0
	}
	return o.Kind != KindList
}

// splitList reads the comma-separated form that `claude plugin configure` stores for a list option.
func splitList(s string) []string {
	var list []string
	for part := range strings.SplitSeq(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			list = append(list, part)
		}
	}
	return list
}

// value converts a saved option into the form config.yaml would give the same key, so the getters treat a bad
// value the same way for both.
func (o Option) value(v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	switch {
	case o.Kind == KindList:
		items := splitList(s)
		list := make([]any, len(items))
		for i, item := range items {
			list[i] = item
		}
		return list
	case o.Kind == KindToggle && s == "on":
		return true
	case o.Kind == KindToggle && s == "off":
		return false
	}
	return v
}

// applyPluginOptions overlays the set options onto the user's values and records which keys they supplied.
func (c *Config) applyPluginOptions(opts *PluginOptions) {
	c.options = opts
	if !opts.Active() {
		return
	}
	if c.fromPlugin == nil {
		c.fromPlugin = map[string]bool{}
	}
	for _, o := range options {
		if v := opts.values[o.Name]; o.set(v) {
			c.setUser(o.Key, o.value(v))
			c.fromPlugin[o.Key] = true
		}
	}
}

// setUser stores a value as if the config file had held it, creating the section when the key is in one.
func (c *Config) setUser(key string, v any) {
	if c.user == nil {
		c.user = map[string]any{}
	}
	section, name, nested := strings.Cut(key, ".")
	if !nested {
		c.user[key] = v
		return
	}
	m := mapOf(c.user[section])
	if m == nil {
		m = map[string]any{}
	}
	m[name] = v
	c.user[section] = m
}

// PluginOptions returns the saved plugin options that were applied, or nil when none were looked for.
func (c *Config) PluginOptions() *PluginOptions {
	return c.options
}

// Source says where the effective value of key came from: the plugin options, the config file or the defaults.
func (c *Config) Source(key string) string {
	switch {
	case c.fromPlugin[key]:
		return SourcePlugin
	case c.lookupUser(key) != nil:
		return SourceFile
	}
	return SourceDefaults
}

// Resolve builds the Config for the environment: the config file over the defaults, with the plugin options saved
// in settings.json over both. A config directory that is missing or unreadable gives the defaults.
func Resolve(getenv func(string) string, pluginDefaults ...string) *Config {
	cfg := New("", pluginDefaults...)
	if path := Path(getenv); path != "" {
		cfg = Load(path, pluginDefaults...)
	}
	cfg.applyPluginOptions(LoadPluginOptions(Dir(getenv)))
	return cfg
}

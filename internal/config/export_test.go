package config

var Parse = parse

func NewWithDefaults(defaults, user map[string]any) *Config {
	return &Config{defaults: defaults, user: user}
}

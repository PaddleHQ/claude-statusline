package config

var Parse = parse

func NewWithDefaults(defaults, user map[string]string) *Config {
	return &Config{defaults: defaults, user: user}
}

package config

var (
	Parse      = parse
	SplitLines = splitLines
)

func NewWithDefaults(defaults, user map[string]any) *Config {
	return &Config{defaults: defaults, user: user}
}

func NewProblem(line int, reason, source string) *Problem {
	return &Problem{Line: line, Reason: reason, source: source}
}

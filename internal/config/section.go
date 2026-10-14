package config

// Section is a view of a Config limited to the keys that start with a prefix, so a plugin reads "effort" and gets "model.effort".
type Section struct {
	cfg    *Config
	prefix string
}

// String returns a non-empty value.
func (s *Section) String(key string) string { return s.cfg.String(s.key(key)) }

// Int returns a non-negative whole number.
func (s *Section) Int(key string) int { return s.cfg.Int(s.key(key)) }

// Bool returns true or false; anything else falls back.
func (s *Section) Bool(key string) bool { return s.cfg.Bool(s.key(key)) }

func (s *Section) key(key string) string {
	if s.prefix == "" {
		return key
	}
	return s.prefix + "." + key
}

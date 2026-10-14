// Package themes embeds the built-in theme files that sit beside it.
package themes

import "embed"

// FS holds the built-in themes, one YAML file per theme name.
//
//go:embed *.yaml
var FS embed.FS

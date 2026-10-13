clear_colors() {
  C_TEXT='' C_MUTED='' C_GOOD='' C_WARN='' C_BAD='' C_MARKER='' RESET=''
}

# Sources themes/<name>.sh and runs theme_<name>; an unknown or missing theme falls back to default.
apply_theme() {
  config_text theme
  theme_name=default
  if valid_name "$CFG_VALUE" && [ -f "$SCRIPT_DIR/themes/$CFG_VALUE.sh" ]; then
    theme_name=$CFG_VALUE
  fi
  # shellcheck source=/dev/null
  . "$SCRIPT_DIR/themes/$theme_name.sh"
  "theme_$theme_name"
  if [ -n "${NO_COLOR:-}" ]; then
    clear_colors
  fi
}

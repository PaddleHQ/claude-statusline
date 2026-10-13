# Sets RENDERER to the function that draws a segment, sourcing its file on demand.
# RENDERER stays empty for an unknown segment.
resolve_segment() {
  RENDERER=''
  valid_name "$1" || return 0
  [ -f "$SCRIPT_DIR/segments/$1.sh" ] || return 0
  # shellcheck source=/dev/null
  . "$SCRIPT_DIR/segments/$1.sh"
  RENDERER="segment_$1"
}

render_line() {
  line=''
  config_text separator
  paint "$C_MUTED" "$CFG_VALUE"
  joiner=" $PAINTED "
  config_text segments
  for segment_name in $CFG_VALUE; do
    resolve_segment "$segment_name"
    [ -n "$RENDERER" ] || continue
    segment_text=$("$RENDERER")
    [ -n "$segment_text" ] || continue
    if [ -z "$line" ]; then
      line=$segment_text
    else
      line="$line$joiner$segment_text"
    fi
  done
  printf '%s' "$line"
}

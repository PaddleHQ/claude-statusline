RESET_TOLERANCE=10

# A small same-window drop is a stale idle session; a drop past the tolerance is a quota reset.
reading_is_newer() {
  [ -n "$1" ] && [ -n "$2" ] || return 1
  [ -n "$3" ] && [ -n "$4" ] || return 0
  if [ "$2" != "$4" ]; then
    [ "$2" -gt "$4" ]
    return
  fi
  [ "$1" -gt "$3" ] || [ "$(( $3 - $1 ))" -gt "$RESET_TOLERANCE" ]
}

load_cache() {
  CACHED_5H_USED='' CACHED_5H_RESET='' CACHED_7D_USED='' CACHED_7D_RESET=''
  [ -f "$CACHE_FILE" ] || return 0
  IFS="$US" read -r CACHED_5H_USED CACHED_5H_RESET CACHED_7D_USED CACHED_7D_RESET < "$CACHE_FILE" || :
  parse_number "$CACHED_5H_USED" ''; CACHED_5H_USED=$NUM
  parse_number "$CACHED_5H_RESET" ''; CACHED_5H_RESET=$NUM
  parse_number "$CACHED_7D_USED" ''; CACHED_7D_USED=$NUM
  parse_number "$CACHED_7D_RESET" ''; CACHED_7D_RESET=$NUM
}

write_cache() {
  mkdir -p "$CACHE_DIR" 2>/dev/null || return 0
  cache_tmp="$CACHE_FILE.$$"
  if { printf '%s\037%s\037%s\037%s\n' "$1" "$2" "$3" "$4" > "$cache_tmp"; } 2>/dev/null; then
    mv -f "$cache_tmp" "$CACHE_FILE" 2>/dev/null || rm -f "$cache_tmp"
  else
    rm -f "$cache_tmp"
  fi
}

# Rate limits are account-global, so display always comes from the merged value.
merge_rate_limits() {
  load_cache
  MERGED_5H_USED=$CACHED_5H_USED MERGED_5H_RESET=$CACHED_5H_RESET
  MERGED_7D_USED=$CACHED_7D_USED MERGED_7D_RESET=$CACHED_7D_RESET
  rate_limits_changed=0
  if reading_is_newer "$IN_5H_USED" "$IN_5H_RESET" "$CACHED_5H_USED" "$CACHED_5H_RESET"; then
    MERGED_5H_USED=$IN_5H_USED MERGED_5H_RESET=$IN_5H_RESET
    rate_limits_changed=1
  fi
  if reading_is_newer "$IN_7D_USED" "$IN_7D_RESET" "$CACHED_7D_USED" "$CACHED_7D_RESET"; then
    MERGED_7D_USED=$IN_7D_USED MERGED_7D_RESET=$IN_7D_RESET
    rate_limits_changed=1
  fi
  if [ "$rate_limits_changed" = 1 ]; then
    write_cache "$MERGED_5H_USED" "$MERGED_5H_RESET" "$MERGED_7D_USED" "$MERGED_7D_RESET"
  fi

  RATE5H_USED='' RATE5H_RESET='' RATE7D_USED='' RATE7D_RESET=''
  if [ -n "$MERGED_5H_USED" ] && [ -n "$MERGED_5H_RESET" ] && [ "$MERGED_5H_RESET" -gt "$NOW" ]; then
    RATE5H_USED=$MERGED_5H_USED RATE5H_RESET=$MERGED_5H_RESET
  fi
  if [ -n "$MERGED_7D_USED" ] && [ -n "$MERGED_7D_RESET" ] && [ "$MERGED_7D_RESET" -gt "$NOW" ]; then
    RATE7D_USED=$MERGED_7D_USED RATE7D_RESET=$MERGED_7D_RESET
  fi
}

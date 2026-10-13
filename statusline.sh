#!/bin/sh
# claude-statusline: segment-based status line for Claude Code.
# Reads the session JSON on stdin and prints a single line. Always exits 0.

set -f

CLAUDE_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
APP_DIR="$CLAUDE_DIR/claude-statusline"
CONFIG_FILE="$APP_DIR/config"
CACHE_DIR="$APP_DIR/cache"
CACHE_FILE="$CACHE_DIR/ratelimits"

US=$(printf '\037')
RESET_TOLERANCE=10
FIVE_HOUR_SECONDS=18000
SEVEN_DAY_SECONDS=604800
MAX_BAR_WIDTH=200

# ============================== Helpers ==============================

trim() {
  TRIMMED=${1#"${1%%[![:space:]]*}"}
  TRIMMED=${TRIMMED%"${TRIMMED##*[![:space:]]}"}
}

# Sets NUM to the value without leading zeros (so arithmetic never reads octal)
# or to the fallback when the value is not a plain number of at most 12 digits.
parse_number() {
  NUM=$2
  case "$1" in
    '' | *[!0-9]*) return 1 ;;
  esac
  parse_value=$1
  while :; do
    case "$parse_value" in
      0?*) parse_value=${parse_value#0} ;;
      *) break ;;
    esac
  done
  [ "${#parse_value}" -le 12 ] || return 1
  NUM=$parse_value
}

valid_name() {
  case "$1" in
    '' | [!a-z]* | *[!a-z0-9_]*) return 1 ;;
  esac
  return 0
}

# A bare name from command -v means a shell function; a path means an external program.
is_function() {
  [ "$(command -v "$1" 2>/dev/null)" = "$1" ]
}

paint() {
  if [ -z "$1" ]; then
    PAINTED=$2
  else
    PAINTED="$1$2$RESET"
  fi
}

# ============================== Config ==============================

# Keys are <name> or <name>.<name>; the validated key becomes part of a variable
# name, while the value is only ever assigned from a variable, never eval'd as text.
config_var_name() {
  CONFIG_VAR=''
  case "$1" in
    *.*.*) return 1 ;;
    *.*)
      valid_name "${1%%.*}" && valid_name "${1#*.}" || return 1
      CONFIG_VAR="CFG__${1%%.*}__${1#*.}"
      ;;
    *)
      valid_name "$1" || return 1
      CONFIG_VAR="CFG__$1"
      ;;
  esac
}

load_config() {
  [ -f "$CONFIG_FILE" ] || return 0
  while IFS= read -r config_line || [ -n "$config_line" ]; do
    trim "$config_line"
    config_line=$TRIMMED
    case "$config_line" in
      '' | '#'*) continue ;;
      *=*) ;;
      *) continue ;;
    esac
    trim "${config_line%%=*}"
    config_key=$TRIMMED
    trim "${config_line#*=}"
    # shellcheck disable=SC2034 # read by the eval below
    config_value=$TRIMMED
    config_var_name "$config_key" || continue
    eval "$CONFIG_VAR=\$config_value"
  done < "$CONFIG_FILE"
}

# Sets CONFIG_RAW to the stored value of a key, empty when unset.
config_raw() {
  CONFIG_RAW=''
  config_var_name "$1" || return 0
  eval "CONFIG_RAW=\${$CONFIG_VAR:-}"
}

# config_text <key> <default>; sets CFG_VALUE.
config_text() {
  config_raw "$1"
  CFG_VALUE=${CONFIG_RAW:-$2}
}

# config_number <key> <default>; sets CFG_VALUE.
# shellcheck disable=SC2329 # only reached through the dynamically resolved segment functions
config_number() {
  config_raw "$1"
  parse_number "$CONFIG_RAW" "$2"
  CFG_VALUE=$NUM
}

# config_bool <key> <default>; sets CFG_VALUE to true or false.
# shellcheck disable=SC2329 # only reached through the dynamically resolved segment functions
config_bool() {
  config_raw "$1"
  case "$CONFIG_RAW" in
    true | false) CFG_VALUE=$CONFIG_RAW ;;
    *) CFG_VALUE=$2 ;;
  esac
}

# ============================== Theme ==============================

theme_default() {
  ESC=$(printf '\033')
  C_TEXT=''
  C_MUTED="${ESC}[2m"
  C_GOOD="${ESC}[32m"
  C_WARN="${ESC}[33m"
  C_BAD="${ESC}[31m"
  C_MARKER="${ESC}[1m"
  RESET="${ESC}[0m"
  G_FULL='█'
  G_PART_1='▏'
  G_PART_2='▎'
  G_PART_3='▍'
  G_PART_4='▌'
  G_PART_5='▋'
  G_PART_6='▊'
  G_PART_7='▉'
  G_EMPTY='░'
  G_MARKER='┃'
}

clear_colors() {
  C_TEXT='' C_MUTED='' C_GOOD='' C_WARN='' C_BAD='' C_MARKER='' RESET=''
}

apply_theme() {
  config_text theme default
  if valid_name "$CFG_VALUE" && is_function "theme_$CFG_VALUE"; then
    "theme_$CFG_VALUE"
  else
    theme_default
  fi
  if [ -n "${NO_COLOR:-}" ]; then
    clear_colors
  fi
}

# ============================== Rate limits ==============================

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

# ============================== Bar ==============================

# Sets BAR_WIDTH: the configured width scaled down for narrow terminals.
# shellcheck disable=SC2329 # only reached through the dynamically resolved segment functions
responsive_width() {
  parse_number "${COLUMNS:-}" 999
  bar_columns=$NUM
  BAR_WIDTH=$1
  if [ "$bar_columns" -lt 120 ]; then
    BAR_WIDTH=$(( BAR_WIDTH / 2 ))
  elif [ "$bar_columns" -lt 160 ]; then
    BAR_WIDTH=$(( BAR_WIDTH * 7 / 10 ))
  fi
  [ "$BAR_WIDTH" -ge 4 ] || BAR_WIDTH=4
  [ "$BAR_WIDTH" -le "$MAX_BAR_WIDTH" ] || BAR_WIDTH=$MAX_BAR_WIDTH
}

# Percentage of the window already elapsed, clamped to 0..100.
# shellcheck disable=SC2329 # only reached through the dynamically resolved segment functions
elapsed_percent() {
  elapsed=$(( NOW - ($1 - $2) ))
  [ "$elapsed" -ge 0 ] || elapsed=0
  [ "$elapsed" -le "$2" ] || elapsed=$2
  ELAPSED_PERCENT=$(( elapsed * 100 / $2 ))
}

# shellcheck disable=SC2329 # only reached through the dynamically resolved segment functions
partial_glyph() {
  case "$1" in
    1) PARTIAL=$G_PART_1 ;;
    2) PARTIAL=$G_PART_2 ;;
    3) PARTIAL=$G_PART_3 ;;
    4) PARTIAL=$G_PART_4 ;;
    5) PARTIAL=$G_PART_5 ;;
    6) PARTIAL=$G_PART_6 ;;
    *) PARTIAL=$G_PART_7 ;;
  esac
}

# make_bar <usage> <target> <width> <warn> <bad> <hard_cap>; sets BAR.
# The fill is coloured by how far usage is from the pace target.
# shellcheck disable=SC2329 # only reached through the dynamically resolved segment functions
make_bar() {
  usage=$1 target=$2 width=$3 warn=$4 bad=$5 hard_cap=$6

  max_eighths=$(( width * 8 ))
  eighths=$(( (usage * max_eighths + 50) / 100 ))
  [ "$eighths" -le "$max_eighths" ] || eighths=$max_eighths
  full_cells=$(( eighths / 8 ))
  remainder=$(( eighths % 8 ))

  target_cell=$(( target * width / 100 ))
  [ "$target_cell" -lt "$width" ] || target_cell=$(( width - 1 ))

  deviation=$(( usage - target ))
  deviation=${deviation#-}
  if [ "$usage" -ge "$hard_cap" ] || [ "$deviation" -gt "$bad" ]; then
    fill_color=$C_BAD
  elif [ "$deviation" -gt "$warn" ]; then
    fill_color=$C_WARN
  else
    fill_color=$C_GOOD
  fi

  BAR=''
  cell=0
  while [ "$cell" -lt "$width" ]; do
    if [ "$cell" -eq "$target_cell" ]; then
      paint "$C_MARKER" "$G_MARKER"
    elif [ "$cell" -lt "$full_cells" ]; then
      paint "$fill_color" "$G_FULL"
    elif [ "$cell" -eq "$full_cells" ] && [ "$remainder" -gt 0 ]; then
      partial_glyph "$remainder"
      paint "$fill_color" "$PARTIAL"
    else
      paint "$C_MUTED" "$G_EMPTY"
    fi
    BAR="$BAR$PAINTED"
    cell=$(( cell + 1 ))
  done
}

# render_rate <segment> <usage> <resets_at> <window_seconds> <default_label> <default_warn> <default_bad>
# shellcheck disable=SC2329 # only reached through the dynamically resolved segment functions
render_rate() {
  [ -n "$2" ] && [ -n "$3" ] || return 0
  config_text "$1.label" "$5"
  rate_label=$CFG_VALUE
  config_number "$1.width" 20
  responsive_width "$CFG_VALUE"
  config_number "$1.warn" "$6"
  rate_warn=$CFG_VALUE
  config_number "$1.bad" "$7"
  rate_bad=$CFG_VALUE
  config_number "$1.hard_cap" 95
  elapsed_percent "$3" "$4"
  make_bar "$2" "$ELAPSED_PERCENT" "$BAR_WIDTH" "$rate_warn" "$rate_bad" "$CFG_VALUE"
  paint "$C_TEXT" "$rate_label: "
  printf '%s%s %s%%' "$PAINTED" "$BAR" "$2"
}

# ============================== Segments ==============================

# shellcheck disable=SC2329 # called via resolve_segment
segment_model() {
  [ -n "$MODEL_NAME" ] || return 0
  model_text=$MODEL_NAME
  config_bool model.effort true
  if [ -n "$EFFORT" ] && [ "$CFG_VALUE" = true ]; then
    model_text="$model_text [$EFFORT]"
  fi
  paint "$C_TEXT" "$model_text"
  printf '%s' "$PAINTED"
}

# shellcheck disable=SC2329 # called via resolve_segment
segment_rate5h() {
  render_rate rate5h "$RATE5H_USED" "$RATE5H_RESET" "$FIVE_HOUR_SECONDS" 5h 20 40
}

# shellcheck disable=SC2329 # called via resolve_segment
segment_rate7d() {
  render_rate rate7d "$RATE7D_USED" "$RATE7D_RESET" "$SEVEN_DAY_SECONDS" 7d 10 20
}

# Maps a segment name to the function that renders it; prints nothing for an unknown name.
resolve_segment() {
  valid_name "$1" || return 0
  if is_function "segment_$1"; then
    printf '%s' "segment_$1"
  fi
}

render_line() {
  line=''
  config_text separator '·'
  paint "$C_MUTED" "$CFG_VALUE"
  joiner=" $PAINTED "
  config_text segments 'model rate5h rate7d'
  for segment_name in $CFG_VALUE; do
    renderer=$(resolve_segment "$segment_name")
    [ -n "$renderer" ] || continue
    segment_text=$("$renderer")
    [ -n "$segment_text" ] || continue
    if [ -z "$line" ]; then
      line=$segment_text
    else
      line="$line$joiner$segment_text"
    fi
  done
  printf '%s' "$line"
}

# ============================== Payload ==============================

# One jq pass; fields are joined by 0x1F so empty ones survive the split.
read_payload() {
  payload_fields=$(printf '%s' "$1" | jq -r '
    def text: if . == null then "" else tostring | gsub("[\u0001-\u001f\u007f]"; "") end;
    def pct: if type == "number" and . >= 0 then round | tostring else "" end;
    def stamp: if type == "number" and . >= 0 then floor | tostring else "" end;
    [
      (.model.display_name | text),
      (.effort.level | text),
      (.rate_limits.five_hour.used_percentage | pct),
      (.rate_limits.five_hour.resets_at | stamp),
      (.rate_limits.seven_day.used_percentage | pct),
      (.rate_limits.seven_day.resets_at | stamp)
    ] | join("\u001f")' 2>/dev/null)
  MODEL_NAME='' EFFORT='' IN_5H_USED='' IN_5H_RESET='' IN_7D_USED='' IN_7D_RESET=''
  IFS="$US" read -r MODEL_NAME EFFORT IN_5H_USED IN_5H_RESET IN_7D_USED IN_7D_RESET <<EOF
$payload_fields
EOF
  parse_number "$IN_5H_USED" ''; IN_5H_USED=$NUM
  parse_number "$IN_5H_RESET" ''; IN_5H_RESET=$NUM
  parse_number "$IN_7D_USED" ''; IN_7D_USED=$NUM
  parse_number "$IN_7D_RESET" ''; IN_7D_RESET=$NUM
}

set_now() {
  if parse_number "${STATUSLINE_NOW:-}" ''; then
    NOW=$NUM
  else
    NOW=$(date +%s)
  fi
}

main() {
  input=$(cat)
  command -v jq >/dev/null 2>&1 || return 0
  load_config
  apply_theme
  set_now
  read_payload "$input"
  merge_rate_limits
  render_line
}

main
exit 0

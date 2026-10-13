MAX_BAR_WIDTH=200

# Sets BAR_WIDTH: the configured width scaled down for narrow terminals.
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
elapsed_percent() {
  elapsed=$(( NOW - ($1 - $2) ))
  [ "$elapsed" -ge 0 ] || elapsed=0
  [ "$elapsed" -le "$2" ] || elapsed=$2
  ELAPSED_PERCENT=$(( elapsed * 100 / $2 ))
}

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

# render_rate <segment> <usage> <resets_at> <window_seconds>
render_rate() {
  [ -n "$2" ] && [ -n "$3" ] || return 0
  config_text "$1.label"
  rate_label=$CFG_VALUE
  config_number "$1.width"
  responsive_width "$CFG_VALUE"
  config_number "$1.warn"
  rate_warn=$CFG_VALUE
  config_number "$1.bad"
  rate_bad=$CFG_VALUE
  config_number "$1.hard_cap"
  elapsed_percent "$3" "$4"
  make_bar "$2" "$ELAPSED_PERCENT" "$BAR_WIDTH" "$rate_warn" "$rate_bad" "$CFG_VALUE"
  paint "$C_TEXT" "$rate_label: "
  printf '%s%s %s%%' "$PAINTED" "$BAR" "$2"
}

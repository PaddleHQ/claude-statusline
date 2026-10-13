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

set_now() {
  if parse_number "${STATUSLINE_NOW:-}" ''; then
    NOW=$NUM
  else
    NOW=$(date +%s)
  fi
}

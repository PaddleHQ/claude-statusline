# Small helpers shared by every part of the status line. Sourced, never run.

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

paint() {
  if [ -z "$1" ]; then
    PAINTED=$2
  else
    PAINTED="$1$2$RESET"
  fi
}

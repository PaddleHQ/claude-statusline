# Two config layers: defaults.conf (DEF__*) and the user's config (CFG__*).
# A user value is used only when valid for the helper asking; otherwise the default is.

# Keys are <name> or <name>.<name>; the validated key becomes part of a variable
# name, while the value is only ever assigned from a variable, never eval'd as text.
config_var_name() {
  CONFIG_VAR=''
  case "$2" in
    *.*.*) return 1 ;;
    *.*)
      valid_name "${2%%.*}" && valid_name "${2#*.}" || return 1
      CONFIG_VAR="$1${2%%.*}__${2#*.}"
      ;;
    *)
      valid_name "$2" || return 1
      CONFIG_VAR="$1$2"
      ;;
  esac
}

# load_config_file <file> <variable prefix>
load_config_file() {
  [ -f "$1" ] || return 0
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
    config_var_name "$2" "$config_key" || continue
    eval "$CONFIG_VAR=\$config_value"
  done < "$1"
}

# Sets CONFIG_RAW to the stored value of a key in one layer, empty when unset.
config_lookup() {
  CONFIG_RAW=''
  config_var_name "$1" "$2" || return 0
  eval "CONFIG_RAW=\${$CONFIG_VAR:-}"
}

# config_text <key>; sets CFG_VALUE.
config_text() {
  config_lookup CFG__ "$1"
  if [ -z "$CONFIG_RAW" ]; then
    config_lookup DEF__ "$1"
  fi
  CFG_VALUE=$CONFIG_RAW
}

# config_number <key>; sets CFG_VALUE.
config_number() {
  config_lookup CFG__ "$1"
  if ! parse_number "$CONFIG_RAW" ''; then
    config_lookup DEF__ "$1"
    parse_number "$CONFIG_RAW" 0
  fi
  CFG_VALUE=$NUM
}

# config_bool <key>; sets CFG_VALUE to true or false.
config_bool() {
  config_lookup CFG__ "$1"
  case "$CONFIG_RAW" in
    true | false) CFG_VALUE=$CONFIG_RAW ;;
    *)
      config_lookup DEF__ "$1"
      CFG_VALUE=$CONFIG_RAW
      ;;
  esac
}

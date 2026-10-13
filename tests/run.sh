#!/bin/sh
# Golden tests: each tests/cases/<name>/ holds input.json and expected, plus optional
# config, cache, env, expected_cache and remove (paths deleted from a copy of the install).
# Escape bytes in output are compared as the text \e.

ROOT=$(cd "$(dirname "$0")/.." && pwd)
SCRIPT="$ROOT/statusline.sh"
CASES="$ROOT/tests/cases"
DEFAULT_NOW=1760000000
ESC=$(printf '\033')
US=$(printf '\037')

pass=0
fail=0

run_case() {
  case_dir=$1
  name=$(basename "$case_dir")
  tmp=$(mktemp -d) || return 1
  app_dir="$tmp/claude-statusline"

  if [ -f "$case_dir/config" ]; then
    mkdir -p "$app_dir"
    sed "s|@TMP@|$tmp|g" "$case_dir/config" > "$app_dir/config"
  fi
  if [ -f "$case_dir/cache" ]; then
    mkdir -p "$app_dir/cache"
    tr '|' "$US" < "$case_dir/cache" > "$app_dir/cache/ratelimits"
  fi

  script=$SCRIPT
  if [ -f "$case_dir/remove" ]; then
    mkdir "$tmp/install"
    cp -R "$ROOT/statusline.sh" "$ROOT/defaults.conf" "$ROOT/lib" "$ROOT/segments" "$ROOT/themes" "$tmp/install/"
    while IFS= read -r removed || [ -n "$removed" ]; do
      [ -z "$removed" ] || rm -f "$tmp/install/$removed"
    done < "$case_dir/remove"
    script=$tmp/install/statusline.sh
  fi

  (
    HOME=$tmp
    CLAUDE_CONFIG_DIR=$tmp
    COLUMNS=200
    STATUSLINE_NOW=$DEFAULT_NOW
    export HOME CLAUDE_CONFIG_DIR COLUMNS STATUSLINE_NOW
    unset NO_COLOR
    if [ -f "$case_dir/env" ]; then
      while IFS= read -r env_line || [ -n "$env_line" ]; do
        case "$env_line" in
          '' | '#'*) continue ;;
          *=*) export "${env_line%%=*}=${env_line#*=}" ;;
        esac
      done < "$case_dir/env"
    fi
    sh "$script" < "$case_dir/input.json" > "$tmp/out.raw"
    echo $? > "$tmp/status"
  )

  problems=''

  [ "$(cat "$tmp/status")" = 0 ] || problems="$problems exit-status"
  [ "$(wc -l < "$tmp/out.raw" | tr -d ' ')" = 0 ] || problems="$problems trailing-newline"
  [ ! -e "$tmp/pwned" ] || problems="$problems injected-file-created"

  actual=$(cat "$tmp/out.raw")
  printf '%s\n' "$actual" | sed "s/$ESC/\\\\e/g" > "$tmp/actual"
  sed "s|@TMP@|$tmp|g" "$case_dir/expected" > "$tmp/expected"
  if ! cmp -s "$tmp/actual" "$tmp/expected"; then
    problems="$problems output"
  fi

  if [ -f "$case_dir/expected_cache" ]; then
    if [ -f "$app_dir/cache/ratelimits" ]; then
      tr "$US" '|' < "$app_dir/cache/ratelimits" > "$tmp/actual_cache"
    else
      : > "$tmp/actual_cache"
    fi
    cmp -s "$tmp/actual_cache" "$case_dir/expected_cache" || problems="$problems cache"
  fi

  if [ -z "$problems" ]; then
    pass=$(( pass + 1 ))
    printf 'ok   %s\n' "$name"
  else
    fail=$(( fail + 1 ))
    printf 'FAIL %s (%s )\n' "$name" "${problems# }"
    case "$problems" in
      *output*)
        echo '--- expected'
        echo '+++ actual'
        diff "$tmp/expected" "$tmp/actual"
        ;;
    esac
    case "$problems" in
      *cache*)
        echo '--- expected cache'
        echo '+++ actual cache'
        diff "$case_dir/expected_cache" "$tmp/actual_cache"
        ;;
    esac
  fi

  rm -rf "$tmp"
}

if ! command -v jq >/dev/null 2>&1; then
  echo 'jq is required to run the tests' >&2
  exit 1
fi

sh "$ROOT/tests/check_defaults.sh" || exit 1

for case_dir in "$CASES"/*/; do
  run_case "${case_dir%/}"
done

printf '\n%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ] || exit 1

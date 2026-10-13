#!/bin/sh
# Every config key the code reads must be in defaults.conf, and every key in
# defaults.conf must be read somewhere.

ROOT=$(cd -- "$(dirname -- "$0")/.." && pwd -P)
cd "$ROOT" || exit 1
tmp=$(mktemp -d) || exit 1
trap 'rm -rf "$tmp"' EXIT

PART='[a-z][a-z0-9_]*'

# Keys named literally, e.g. config_number rate5h.width
grep -rhoE "config_(text|number|bool) $PART(\\.$PART)?" lib segments themes statusline.sh |
  awk '{ print $2 }' > "$tmp/used"

# Keys built from a segment name, e.g. config_text "$1.label" inside render_rate
grep -rhoE "config_(text|number|bool) \"\\\$1\\.$PART\"" lib segments themes |
  sed 's/.*[$]1\.//; s/"$//' > "$tmp/suffixes"
grep -rhoE "render_rate $PART" segments | awk '{ print $2 }' > "$tmp/prefixes"
while IFS= read -r prefix; do
  while IFS= read -r suffix; do
    printf '%s.%s\n' "$prefix" "$suffix"
  done < "$tmp/suffixes"
done < "$tmp/prefixes" >> "$tmp/used"

sed -n 's/^[[:space:]]*\([a-z][a-z0-9_.]*\)[[:space:]]*=.*/\1/p' defaults.conf > "$tmp/defined"

sort -u "$tmp/used" > "$tmp/used.sorted"
sort "$tmp/defined" > "$tmp/defined.sorted"

status=0
if [ "$(uniq -d "$tmp/defined.sorted" | wc -l)" -ne 0 ]; then
  echo 'keys defined more than once in defaults.conf:'
  uniq -d "$tmp/defined.sorted" | sed 's/^/  /'
  status=1
fi
sort -u "$tmp/defined.sorted" > "$tmp/defined.unique"

missing=$(comm -23 "$tmp/used.sorted" "$tmp/defined.unique")
if [ -n "$missing" ]; then
  echo 'keys read by the code but missing from defaults.conf:'
  printf '%s\n' "$missing" | sed 's/^/  /'
  status=1
fi
unused=$(comm -13 "$tmp/used.sorted" "$tmp/defined.unique")
if [ -n "$unused" ]; then
  echo 'keys in defaults.conf that nothing reads:'
  printf '%s\n' "$unused" | sed 's/^/  /'
  status=1
fi

[ "$status" -ne 0 ] || echo "defaults check: $(wc -l < "$tmp/defined.unique" | tr -d ' ') keys ok"
exit "$status"

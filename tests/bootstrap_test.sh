#!/bin/sh
# Tests for hooks/bootstrap.sh. Every test runs under a temporary HOME.
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PASS=0
FAIL=0
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

case "$(uname -s)" in
  Darwin) OS=darwin ;;
  Linux) OS=linux ;;
  *) echo "unsupported platform; skipping all tests"; exit 0 ;;
esac
case "$(uname -m)" in
  arm64 | aarch64) ARCH=arm64 ;;
  x86_64 | amd64) ARCH=amd64 ;;
  *) echo "unsupported architecture; skipping all tests"; exit 0 ;;
esac

REAL_CLAUDE="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
real_state() {
  ls -ld "$REAL_CLAUDE/claude-statusline" 2>&1
  cksum "$REAL_CLAUDE/settings.json" 2>&1
}
REAL_BEFORE=$(real_state)

sha256_of() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{ print $1 }'
  else
    sha256sum "$1" | awk '{ print $1 }'
  fi
}

ok() {
  PASS=$((PASS + 1))
  echo "ok   - $NAME"
}

bad() {
  FAIL=$((FAIL + 1))
  echo "FAIL - $NAME: $1"
}

# Builds a fake release for version $1 and a plugin root pinned to $2.
make_release() {
  v="$1"
  d="$T/build-$v"
  mkdir -p "$d" "$T/releases/v$v"
  cat > "$d/claude-statusline" <<STUB
#!/bin/sh
[ "\$1" = --version ] && echo "claude-statusline $v (abc, 2026-01-01)"
exit 0
STUB
  chmod 755 "$d/claude-statusline"
  archive="claude-statusline_${v}_${OS}_${ARCH}.tar.gz"
  tar -czf "$T/releases/v$v/$archive" -C "$d" claude-statusline
  printf '%s  %s\n' "$(sha256_of "$T/releases/v$v/$archive")" "$archive" >> "$T/checksums.all"
}

pin() {
  mkdir -p "$T/plugin/release"
  printf '%s\n' "$1" > "$T/plugin/release/version"
  if [ -n "${2:-}" ]; then
    grep -F "_${1}_" "$T/checksums.all" > "$T/plugin/release/checksums.txt"
  else
    : > "$T/plugin/release/checksums.txt"
  fi
}

new_env() {
  T="$WORK/$1"
  mkdir -p "$T/home" "$T/claude" "$T/plugin"
  : > "$T/checksums.all"
  cp -R "$ROOT/hooks" "$T/plugin/hooks"
  BIN="$T/claude/claude-statusline/bin/claude-statusline"
  SETTINGS="$T/claude/settings.json"
}

run_hook() {
  HOME="$T/home" CLAUDE_CONFIG_DIR="$T/claude" CLAUDE_PLUGIN_ROOT="$T/plugin" \
    CLAUDE_STATUSLINE_DOWNLOAD_BASE="file://$T/releases" \
    sh "$T/plugin/hooks/bootstrap.sh" > "$T/out" 2> "$T/err"
  RC=$?
}

assert_clean_exit() {
  [ "$RC" = 0 ] || bad "exit code $RC"
  [ ! -s "$T/err" ] || bad "stderr not empty: $(cat "$T/err")"
}

assert_silent() {
  [ ! -s "$T/out" ] || bad "expected no output, got: $(cat "$T/out")"
}

message() {
  jq -er '.systemMessage' "$T/out" 2>/dev/null
}

assert_message_matches() {
  msg=$(message) || { bad "output is not a systemMessage JSON: $(cat "$T/out")"; return; }
  printf '%s' "$msg" | grep -Eq "$1" || bad "message '$msg' does not match $1"
  [ "$(wc -l < "$T/out" | tr -d ' ')" = 1 ] || bad "expected exactly one line of output"
}

finish() {
  if [ "$FAIL" = "$FAIL_START" ]; then ok; fi
}

start() {
  NAME="$1"
  FAIL_START="$FAIL"
  new_env "$2"
}

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required to run these tests" >&2
  exit 1
fi

start "empty pin is a silent no-op" t1
pin ""
run_hook
assert_clean_exit
assert_silent
[ ! -e "$T/claude/claude-statusline" ] || bad "created the app dir"
[ ! -e "$SETTINGS" ] || bad "created settings.json"
finish

start "fresh install wires statusLine, backup and sentinel" t2
make_release 1.2.3
pin 1.2.3 y
printf '{"theme":"dark"}\n' > "$SETTINGS"
run_hook
assert_clean_exit
assert_message_matches 'v1\.2\.3 installed and enabled'
[ -x "$BIN" ] || bad "binary not installed"
[ -e "$T/claude/claude-statusline/.bootstrapped" ] || bad "no sentinel"
[ "$(cat "$SETTINGS.bak")" = '{"theme":"dark"}' ] || bad "backup differs from original"
[ "$(jq -r '.statusLine.command' "$SETTINGS")" = "$BIN" ] || bad "statusLine.command wrong"
[ "$(jq -r '.statusLine.refreshInterval' "$SETTINGS")" = 60 ] || bad "refreshInterval wrong"
[ "$(jq -r '.statusLine.type' "$SETTINGS")" = command ] || bad "type wrong"
[ "$(jq -r '.theme' "$SETTINGS")" = dark ] || bad "existing settings lost"
for left in "$T"/claude/claude-statusline/tmp.*; do [ ! -e "$left" ] || bad "temp dir left behind"; done
finish

start "second run is silent and does not touch the network" t3
make_release 1.2.3
pin 1.2.3 y
run_hook
cp "$SETTINGS" "$T/settings.after1"
rm -rf "$T/releases"
run_hook
assert_clean_exit
assert_silent
cmp -s "$SETTINGS" "$T/settings.after1" || bad "settings changed on second run"
finish

start "fresh install with no settings.json creates one" t4
make_release 1.2.3
pin 1.2.3 y
run_hook
assert_clean_exit
[ "$(jq -r '.statusLine.command' "$SETTINGS")" = "$BIN" ] || bad "statusLine not set"
finish

start "existing user statusLine is left untouched" t5
make_release 1.2.3
pin 1.2.3 y
printf '{"statusLine":{"type":"command","command":"/usr/bin/mine"}}\n' > "$SETTINGS"
cp "$SETTINGS" "$T/settings.orig"
run_hook
assert_clean_exit
assert_message_matches 'already have a statusLine'
cmp -s "$SETTINGS" "$T/settings.orig" || bad "settings.json was modified"
[ ! -e "$SETTINGS.bak" ] || bad "unexpected backup"
[ -e "$T/claude/claude-statusline/.bootstrapped" ] || bad "no sentinel"
[ -x "$BIN" ] || bad "binary not installed"
finish

start "checksum mismatch is not installed" t6
make_release 1.2.3
pin 1.2.3 y
printf 'tampered' >> "$T/releases/v1.2.3/claude-statusline_1.2.3_${OS}_${ARCH}.tar.gz"
run_hook
assert_clean_exit
assert_message_matches 'FAILED checksum verification and was NOT installed'
[ ! -e "$BIN" ] || bad "binary was installed"
[ ! -e "$SETTINGS" ] || bad "settings.json was created"
[ ! -e "$T/claude/claude-statusline/.bootstrapped" ] || bad "sentinel created"
finish

start "missing checksum line gives a message" t7
make_release 1.2.3
pin 1.2.3
run_hook
assert_clean_exit
assert_message_matches 'no checksum is pinned'
[ ! -e "$BIN" ] || bad "binary was installed"
finish

start "version bump replaces the binary and leaves settings alone" t8
make_release 1.2.3
make_release 1.2.4
pin 1.2.3 y
run_hook
cp "$SETTINGS" "$T/settings.after1"
pin 1.2.4 y
run_hook
assert_clean_exit
assert_message_matches 'updated to v1\.2\.4'
[ "$("$BIN" --version | awk '{ print $2 }')" = 1.2.4 ] || bad "binary not upgraded"
cmp -s "$SETTINGS" "$T/settings.after1" || bad "settings changed on upgrade"
finish

start "invalid version string gives a message" t9
pin "1.2" y
run_hook
assert_clean_exit
assert_message_matches 'invalid pinned version'
[ ! -e "$T/claude/claude-statusline" ] || bad "created the app dir"
finish

start "failed upgrade keeps the previous binary" t10
make_release 1.2.3
make_release 1.2.4
pin 1.2.3 y
run_hook
printf 'tampered' >> "$T/releases/v1.2.4/claude-statusline_1.2.4_${OS}_${ARCH}.tar.gz"
pin 1.2.4 y
run_hook
assert_clean_exit
assert_message_matches 'FAILED checksum verification'
[ "$("$BIN" --version | awk '{ print $2 }')" = 1.2.3 ] || bad "previous binary was replaced"
finish

start "invalid settings.json is left untouched" t11
make_release 1.2.3
pin 1.2.3 y
printf 'not json' > "$SETTINGS"
run_hook
assert_clean_exit
assert_message_matches 'not valid JSON'
[ "$(cat "$SETTINGS")" = 'not json' ] || bad "settings.json was modified"
[ ! -e "$T/claude/claude-statusline/.bootstrapped" ] || bad "sentinel created"
finish

start "no jq on PATH prints the settings snippet" t12
make_release 1.2.3
pin 1.2.3 y
printf '{"theme":"dark"}\n' > "$SETTINGS"
NOJQ="$T/nojq"
mkdir -p "$NOJQ"
for tool in sh awk sed grep tr cat cp mv rm mkdir mktemp tar tail head cut uname dirname chmod touch curl shasum sha256sum wc ls date; do
  p=$(command -v "$tool" 2>/dev/null) && ln -s "$p" "$NOJQ/$tool"
done
PATH="$NOJQ" HOME="$T/home" CLAUDE_CONFIG_DIR="$T/claude" CLAUDE_PLUGIN_ROOT="$T/plugin" \
  CLAUDE_STATUSLINE_DOWNLOAD_BASE="file://$T/releases" \
  "$NOJQ/sh" "$T/plugin/hooks/bootstrap.sh" > "$T/out" 2> "$T/err"
RC=$?
assert_clean_exit
grep -q 'jq was not found' "$T/out" || bad "missing jq message: $(cat "$T/out")"
jq -e '.systemMessage' "$T/out" >/dev/null || bad "output is not valid JSON"
[ "$(cat "$SETTINGS")" = '{"theme":"dark"}' ] || bad "settings.json was modified"
[ -e "$T/claude/claude-statusline/.bootstrapped" ] || bad "no sentinel"
finish

NAME="real Claude config dir untouched"
if [ "$(real_state)" = "$REAL_BEFORE" ]; then ok; else bad "state under $REAL_CLAUDE changed"; fi

echo
echo "$PASS passed, $FAIL failed"
[ "$FAIL" = 0 ]

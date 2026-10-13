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

# The stand-in binary answers --version and records and echoes its setup arguments.
make_release() {
  v="$1"
  d="$T/build-$v"
  mkdir -p "$d" "$T/releases/v$v"
  cat > "$d/claude-statusline" <<STUB
#!/bin/sh
case "\$1" in
  --version) echo "claude-statusline $v (abc, 2026-01-01)" ;;
  setup) echo "\$*" >> "\$STUB_LOG"; echo "{\"systemMessage\":\"stub \$2\"}" ;;
esac
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

start() {
  NAME="$1"
  FAIL_START="$FAIL"
  T="$WORK/$2"
  mkdir -p "$T/home" "$T/claude" "$T/plugin"
  : > "$T/checksums.all"
  : > "$T/setup.log"
  cp -R "$ROOT/hooks" "$T/plugin/hooks"
  BIN="$T/claude/claude-statusline/bin/claude-statusline"
}

run_hook() {
  HOME="$T/home" CLAUDE_CONFIG_DIR="$T/claude" CLAUDE_PLUGIN_ROOT="$T/plugin" \
    CLAUDE_STATUSLINE_DOWNLOAD_BASE="file://$T/releases" STUB_LOG="$T/setup.log" \
    sh "$T/plugin/hooks/bootstrap.sh" > "$T/out" 2> "$T/err"
  RC=$?
  [ "$RC" = 0 ] || bad "exit code $RC"
  [ ! -s "$T/err" ] || bad "stderr not empty: $(cat "$T/err")"
}

assert_setup_events() {
  [ "$(cat "$T/setup.log")" = "$1" ] || bad "setup calls were '$(cat "$T/setup.log")', want '$1'"
}

assert_silent() {
  [ ! -s "$T/out" ] || bad "expected no output, got: $(cat "$T/out")"
}

assert_message() {
  grep -q "^{\"systemMessage\":\".*$1.*\"}\$" "$T/out" || bad "output '$(cat "$T/out")' lacks message matching $1"
  [ "$(wc -l < "$T/out" | tr -d ' ')" = 1 ] || bad "expected exactly one line of output"
}

assert_not_installed() {
  [ ! -e "$BIN" ] || bad "binary was installed"
  assert_setup_events ""
}

finish() {
  if [ "$FAIL" = "$FAIL_START" ]; then ok; fi
}

start "empty pin is a silent no-op" t1
pin ""
run_hook
assert_silent
assert_setup_events ""
[ ! -e "$T/claude/claude-statusline" ] || bad "created the app dir"
finish

start "fresh install runs setup with the installed event" t2
make_release 1.2.3
pin 1.2.3 y
run_hook
assert_setup_events "setup --event=installed"
assert_message 'stub --event=installed'
[ -x "$BIN" ] || bad "binary not installed"
for left in "$T"/claude/claude-statusline/tmp.*; do [ ! -e "$left" ] || bad "temp dir left behind"; done
finish

start "second run makes no network request and runs setup with the current event" t3
make_release 1.2.3
pin 1.2.3 y
run_hook
rm -rf "$T/releases"
: > "$T/setup.log"
run_hook
assert_setup_events "setup --event=current"
finish

start "checksum mismatch is not installed" t4
make_release 1.2.3
pin 1.2.3 y
printf 'tampered' >> "$T/releases/v1.2.3/claude-statusline_1.2.3_${OS}_${ARCH}.tar.gz"
run_hook
assert_message 'FAILED checksum verification and was NOT installed'
assert_not_installed
finish

start "missing checksum line gives a message" t5
make_release 1.2.3
pin 1.2.3
run_hook
assert_message 'no checksum is pinned'
assert_not_installed
finish

start "version bump replaces the binary and runs setup with the updated event" t6
make_release 1.2.3
make_release 1.2.4
pin 1.2.3 y
run_hook
: > "$T/setup.log"
pin 1.2.4 y
run_hook
assert_setup_events "setup --event=updated"
[ "$("$BIN" --version | awk '{ print $2 }')" = 1.2.4 ] || bad "binary not upgraded"
finish

start "failed upgrade keeps the previous binary" t7
make_release 1.2.3
make_release 1.2.4
pin 1.2.3 y
run_hook
: > "$T/setup.log"
printf 'tampered' >> "$T/releases/v1.2.4/claude-statusline_1.2.4_${OS}_${ARCH}.tar.gz"
pin 1.2.4 y
run_hook
assert_message 'FAILED checksum verification'
assert_setup_events ""
[ "$("$BIN" --version | awk '{ print $2 }')" = 1.2.3 ] || bad "previous binary was replaced"
finish

start "invalid version string gives a message" t8
pin "1.2" y
run_hook
assert_message 'invalid pinned version'
assert_setup_events ""
[ ! -e "$T/claude/claude-statusline" ] || bad "created the app dir"
finish

start "unreachable download gives a message and installs nothing" t9
make_release 1.2.3
pin 1.2.3 y
rm -rf "$T/releases"
run_hook
assert_message 'downloading v1.2.3 failed'
assert_not_installed
finish

NAME="real Claude config dir untouched"
if [ "$(real_state)" = "$REAL_BEFORE" ]; then ok; else bad "state under $REAL_CLAUDE changed"; fi

echo
echo "$PASS passed, $FAIL failed"
[ "$FAIL" = 0 ]

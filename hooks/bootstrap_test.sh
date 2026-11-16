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

SH=$(command -v sh)

# A PATH holding only symlinks to the tools the hook needs, so the machine's own gh is never found.
TOOLS="$WORK/tools"
mkdir -p "$TOOLS"
for tool in awk cat chmod curl date dirname find grep gzip mkdir mktemp mv rm sed sh sha256sum shasum tar tr uname wc; do
  found=$(command -v "$tool" 2>/dev/null) && case "$found" in /*) ln -s "$found" "$TOOLS/$tool" ;; esac
done

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

write_stub() {
  mkdir -p "$(dirname "$1")"
  cat > "$1" <<STUB
#!/bin/sh
case "\$1" in
  --version) echo "claude-statusline $2 (abc, 2026-01-01)" ;;
  setup) echo "\$*" >> "\$STUB_LOG"; echo "{\"systemMessage\":\"stub \$2\"}" ;;
esac
exit 0
STUB
  chmod 755 "$1"
}

make_release() {
  v="$1"
  d="$T/build-$v"
  write_stub "$d/claude-statusline" "$v"
  mkdir -p "$T/releases/v$v"
  archive="claude-statusline_${v}_${OS}_${ARCH}.tar.gz"
  tar -czf "$T/releases/v$v/$archive" -C "$d" claude-statusline
  printf '%s  %s\n' "$(sha256_of "$T/releases/v$v/$archive")" "$archive" > "$T/releases/v$v/checksums.txt"
  printf '%064d  claude-statusline_%s_other_arch.tar.gz\n' 0 "$v" >> "$T/releases/v$v/checksums.txt"
  printf '{}\n' > "$T/releases/v$v/provenance.jsonl"
}

set_latest() {
  printf '%s\n' "$1" > "$T/latest"
}

install_stub() {
  write_stub "$BIN" "$1"
  age_check "${2:-90000}"
}

age_check() {
  mkdir -p "$APP"
  printf '%s\n' "$(($(date +%s) - $1))" > "$CHECK"
}

stub_gh() {
  mkdir -p "$T/gh"
  cat > "$T/gh/gh" <<STUB
#!/bin/sh
echo "\$*" >> "$T/gh.log"
case "\$1:\$2:$1" in
  attestation:*:oldgh) exit 1 ;;
  attestation:verify:badattest) exit 1 ;;
esac
exit 0
STUB
  chmod 755 "$T/gh/gh"
}

start() {
  DATA_MODE=0
  NAME="$1"
  FAIL_START="$FAIL"
  T="$WORK/$2"
  mkdir -p "$T/home" "$T/claude" "$T/plugin" "$T/gh"
  : > "$T/setup.log"
  : > "$T/gh.log"
  cp -R "$ROOT/hooks" "$T/plugin/hooks"
  set_app_dir "$T/claude/claude-statusline"
}

set_app_dir() {
  APP="$1"
  BIN="$APP/bin/claude-statusline"
  CHECK="$APP/last-check"
  NOTICE="$APP/.no-attestation-notice"
}

use_plugin_data() {
  DATA_MODE=1
  set_app_dir "$T/claude/plugins/data/statusline-claude-statusline"
}

run_hook() {
  data=""
  [ "$DATA_MODE" = 0 ] || data="$APP"
  PATH="$T/gh:$TOOLS" HOME="$T/home" CLAUDE_CONFIG_DIR="$T/claude" CLAUDE_PLUGIN_ROOT="$T/plugin" CLAUDE_PLUGIN_DATA="$data" \
    CLAUDE_STATUSLINE_DOWNLOAD_BASE="file://$T/releases" CLAUDE_STATUSLINE_LATEST_URL="file://$T/latest" \
    STUB_LOG="$T/setup.log" "$SH" "$T/plugin/hooks/bootstrap.sh" > "$T/out" 2> "$T/err"
  RC=$?
  [ "$RC" = 0 ] || bad "exit code $RC"
  [ ! -s "$T/err" ] || bad "stderr not empty: $(cat "$T/err")"
}

installed_version() {
  "$BIN" --version | awk '{ print $2 }'
}

assert_version() {
  [ "$(installed_version)" = "$1" ] || bad "installed version is $(installed_version), want $1"
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

assert_check_fresh() {
  [ "$(($(date +%s) - $(cat "$CHECK")))" -lt 300 ] || bad "last-check was not updated"
}

assert_no_temp_left() {
  for left in "$APP"/tmp.*; do [ ! -e "$left" ] || bad "temp dir left behind"; done
}

finish() {
  if [ "$FAIL" = "$FAIL_START" ]; then ok; fi
}

start "fresh install downloads the latest release and runs setup with the installed event" t1
make_release 1.2.3
set_latest v1.2.3
stub_gh login
run_hook
assert_setup_events "setup --event=installed"
assert_message 'stub --event=installed'
assert_version 1.2.3
assert_check_fresh
assert_no_temp_left
finish

start "within 24 hours there is no network request and setup runs with the current event" t2
make_release 1.2.3
set_latest v1.2.3
stub_gh login
run_hook
rm -rf "$T/releases" "$T/latest"
: > "$T/setup.log"
run_hook
assert_setup_events "setup --event=current"
assert_version 1.2.3
finish

start "after 24 hours with the same version nothing is downloaded and last-check is updated" t3
make_release 1.2.3
set_latest v1.2.3
install_stub 1.2.3
rm -rf "$T/releases/v1.2.3"
run_hook
assert_setup_events "setup --event=current"
assert_check_fresh
assert_version 1.2.3
finish

start "a newer release replaces the binary and runs setup with the updated event" t4
make_release 1.2.4
set_latest v1.2.4
stub_gh login
install_stub 1.2.3
run_hook
assert_setup_events "setup --event=updated"
assert_version 1.2.4
assert_check_fresh
finish

start "an older release is never installed over a newer binary" t5
make_release 1.2.3
set_latest v1.2.3
install_stub 1.3.0
run_hook
assert_setup_events "setup --event=current"
assert_version 1.3.0
assert_check_fresh
finish

start "a numeric comparison is used rather than a string one" t6
make_release 1.10.0
set_latest v1.10.0
stub_gh login
install_stub 1.9.9
run_hook
assert_version 1.10.0
install_stub 1.10.0
make_release 1.9.0
set_latest v1.9.0
run_hook
assert_version 1.10.0
finish

start "a development version counts as older" t7
make_release 0.1.0
set_latest v0.1.0
stub_gh login
install_stub 0.0.0-stack
run_hook
assert_setup_events "setup --event=updated"
assert_version 0.1.0
finish

start "a checksum mismatch is not installed" t8
make_release 1.2.3
set_latest v1.2.3
printf 'tampered' >> "$T/releases/v1.2.3/claude-statusline_1.2.3_${OS}_${ARCH}.tar.gz"
run_hook
assert_message 'FAILED checksum verification and was NOT installed'
assert_not_installed
finish

start "a checksums file without the platform line gives a message" t9
make_release 1.2.3
set_latest v1.2.3
grep -v "_${OS}_${ARCH}" "$T/releases/v1.2.3/checksums.txt" > "$T/c" && mv "$T/c" "$T/releases/v1.2.3/checksums.txt"
run_hook
assert_message 'has no checksum for'
assert_not_installed
finish

start "a missing checksums file gives a message" t10
make_release 1.2.3
set_latest v1.2.3
rm "$T/releases/v1.2.3/checksums.txt"
run_hook
assert_message 'downloading the checksums'
assert_not_installed
finish

start "a failed attestation is not installed" t11
make_release 1.2.3
set_latest v1.2.3
stub_gh badattest
run_hook
assert_message 'provenance verification of v1.2.3 FAILED'
assert_not_installed
grep -q '^attestation verify .* --repo PaddleHQ/claude-statusline$' "$T/gh.log" || bad "gh attestation verify was not called with the repo"
finish

start "a failed attestation keeps the previous binary" t12
make_release 1.2.4
set_latest v1.2.4
stub_gh badattest
install_stub 1.2.3
run_hook
assert_message 'provenance verification'
assert_setup_events ""
assert_version 1.2.3
finish

start "a passing attestation installs without the notice" t13
make_release 1.2.3
set_latest v1.2.3
stub_gh login
run_hook
assert_version 1.2.3
assert_message 'stub --event=installed'
if grep -q 'provenance' "$T/out"; then bad "notice shown although gh verified the build"; fi
[ ! -e "$NOTICE" ] || bad "sentinel written although gh verified the build"
grep -q '^attestation verify .* --bundle .*/provenance.jsonl ' "$T/gh.log" || bad "gh attestation verify was not run against the release's bundle"
grep -q ' --signer-workflow PaddleHQ/claude-statusline/.github/workflows/release-binaries.yml ' "$T/gh.log" || bad "gh attestation verify did not pin the release workflow"
finish

start "without gh it installs on the checksum and shows the notice once" t14
make_release 1.2.3
make_release 1.2.4
set_latest v1.2.3
rm -rf "$T/gh"
run_hook
assert_version 1.2.3
assert_message 'stub --event=installed.*checksum check only'
[ -e "$NOTICE" ] || bad "sentinel not written"
set_latest v1.2.4
age_check 90000
: > "$T/setup.log"
run_hook
assert_version 1.2.4
assert_setup_events "setup --event=updated"
if grep -q 'checksum check only' "$T/out"; then bad "notice shown a second time"; fi
finish

start "a release without a provenance bundle is not installed when gh is present" t15c
make_release 1.2.3
set_latest v1.2.3
rm "$T/releases/v1.2.3/provenance.jsonl"
stub_gh login
run_hook
assert_message 'downloading the provenance bundle'
assert_not_installed
finish

start "gh without the attestation command is treated as no gh, not as a failed verification" t15b
make_release 1.2.3
set_latest v1.2.3
stub_gh oldgh
run_hook
assert_version 1.2.3
assert_message 'checksum check only'
if grep -q '^attestation verify' "$T/gh.log"; then bad "verify was attempted on a gh without the attestation command"; fi
finish

start "a network failure with a binary installed is silent and last-check is not advanced" t16
rm -f "$T/latest"
install_stub 1.2.3
before=$(cat "$CHECK")
run_hook
assert_setup_events "setup --event=current"
if grep -q 'systemMessage":"claude-statusline' "$T/out"; then bad "expected no claude-statusline message"; fi
[ "$(cat "$CHECK")" = "$before" ] || bad "last-check was advanced"
assert_version 1.2.3
finish

start "a failed download with a binary installed is silent and last-check is not advanced" t17
make_release 1.2.4
set_latest v1.2.4
rm -rf "$T/releases/v1.2.4"
install_stub 1.2.3
before=$(cat "$CHECK")
run_hook
if grep -q 'systemMessage":"claude-statusline' "$T/out"; then bad "expected no claude-statusline message"; fi
[ "$(cat "$CHECK")" = "$before" ] || bad "last-check was advanced"
assert_version 1.2.3
assert_no_temp_left
finish

start "a network failure with nothing installed gives a message" t18
rm -f "$T/latest"
run_hook
assert_message 'could not find the latest release'
assert_not_installed
finish

start "an invalid latest tag installs nothing" t19
make_release 1.2.3
set_latest "v1.2"
run_hook
assert_message 'not a valid version'
assert_not_installed
set_latest "../../etc/passwd"
run_hook
assert_message 'not a valid version'
assert_not_installed
finish

start "a latest tag with a pre-release suffix installs nothing" t20
make_release 1.2.3
set_latest "v1.2.3-rc1"
run_hook
assert_message 'not a valid version'
assert_not_installed
finish

start "with CLAUDE_PLUGIN_DATA the binary, last-check and notice live in the data directory and nothing in the legacy one" t21
use_plugin_data
make_release 1.2.3
set_latest v1.2.3
rm -rf "$T/gh"
run_hook
assert_setup_events "setup --event=installed"
assert_version 1.2.3
assert_check_fresh
assert_no_temp_left
[ -e "$NOTICE" ] || bad "notice marker not written in the data directory"
[ ! -e "$T/claude/claude-statusline" ] || bad "the legacy directory was created"
finish

start "with CLAUDE_PLUGIN_DATA an update replaces the binary in the data directory" t22
use_plugin_data
make_release 1.2.4
set_latest v1.2.4
stub_gh login
install_stub 1.2.3
run_hook
assert_setup_events "setup --event=updated"
assert_version 1.2.4
[ ! -e "$T/claude/claude-statusline" ] || bad "the legacy directory was created"
finish

start "with CLAUDE_PLUGIN_DATA a recent check makes no network request" t23
use_plugin_data
install_stub 1.2.3 60
before=$(cat "$CHECK")
run_hook
assert_setup_events "setup --event=current"
[ "$(cat "$CHECK")" = "$before" ] || bad "last-check was advanced, so a check was made"
finish

start "without CLAUDE_PLUGIN_DATA the legacy layout is used" t24
make_release 1.2.3
set_latest v1.2.3
stub_gh login
run_hook
assert_version 1.2.3
[ -x "$T/claude/claude-statusline/bin/claude-statusline" ] || bad "binary not in the legacy directory"
[ ! -e "$T/claude/plugins" ] || bad "the plugin data directory was created"
finish

run_run() {
  OUT=$(HOME="$T/home" CLAUDE_CONFIG_DIR="$T/claude" CLAUDE_PLUGIN_DATA="$1" STUB_LOG="$T/setup.log" "$SH" "$T/plugin/hooks/run.sh" "$2" 2>&1)
}

start "run.sh runs the binary in CLAUDE_PLUGIN_DATA with its arguments" t25
write_stub "$T/data/bin/claude-statusline" 1.2.3
write_stub "$T/claude/claude-statusline/bin/claude-statusline" 0.1.0
run_run "$T/data" --version
case "$OUT" in "claude-statusline 1.2.3"*) ;; *) bad "ran '$OUT'" ;; esac
finish

start "run.sh finds the plugin data directory from the config directory when CLAUDE_PLUGIN_DATA is not set" t26
write_stub "$T/claude/plugins/data/statusline-claude-statusline/bin/claude-statusline" 1.2.3
run_run "" --version
case "$OUT" in "claude-statusline 1.2.3"*) ;; *) bad "ran '$OUT'" ;; esac
finish

start "run.sh falls back to the legacy install" t27
write_stub "$T/claude/claude-statusline/bin/claude-statusline" 0.1.0
run_run "" --version
case "$OUT" in "claude-statusline 0.1.0"*) ;; *) bad "ran '$OUT'" ;; esac
finish

start "run.sh says NOT-INSTALLED when there is no binary" t28
run_run "" --version
[ "$OUT" = "NOT-INSTALLED" ] || bad "printed '$OUT'"
finish

start "run.sh reports a non-zero exit status on its last line" t29
mkdir -p "$T/data/bin"
printf '#!/bin/sh\necho "Config file: x"\nexit 1\n' > "$T/data/bin/claude-statusline"
chmod +x "$T/data/bin/claude-statusline"
run_run "$T/data" config-check
[ "$OUT" = "$(printf 'Config file: x\nexit status: 1')" ] || bad "printed '$OUT'"
finish

NAME="real Claude config dir untouched"
if [ "$(real_state)" = "$REAL_BEFORE" ]; then ok; else bad "state under $REAL_CLAUDE changed"; fi

echo
echo "$PASS passed, $FAIL failed"
[ "$FAIL" = 0 ]

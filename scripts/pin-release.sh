#!/bin/sh
# Maintainer script: pins the plugin to a published release after verifying its
# build attestations and checksums. Writes nothing unless every check passes.
set -eu

REPO="PaddleHQ/claude-statusline"

if [ $# -ne 1 ]; then
  echo "usage: $0 <version>   (for example 0.1.0, without the leading v)" >&2
  exit 2
fi
V="$1"
if ! printf '%s' "$V" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "error: version must look like 1.2.3, got: $V" >&2
  exit 2
fi
if ! command -v gh >/dev/null 2>&1; then
  echo "error: gh is required (https://cli.github.com)" >&2
  exit 2
fi
if ! command -v shasum >/dev/null 2>&1 && ! command -v sha256sum >/dev/null 2>&1; then
  echo "error: shasum or sha256sum is required" >&2
  exit 2
fi

RELEASE_DIR="$(cd "$(dirname "$0")/.." && pwd)/release"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

sha256_of() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{ print $1 }'
  else
    sha256sum "$1" | awk '{ print $1 }'
  fi
}

ARCHIVES=""
for os in darwin linux; do
  for arch in arm64 amd64; do
    ARCHIVES="$ARCHIVES claude-statusline_${V}_${os}_${arch}.tar.gz"
  done
done

echo "Downloading v$V from $REPO"
set -- --pattern checksums.txt
for a in $ARCHIVES; do
  set -- "$@" --pattern "$a"
done
gh release download "v$V" --repo "$REPO" --dir "$TMP" "$@"

: > "$TMP/pinned.txt"
for a in $ARCHIVES; do
  echo "Verifying attestation for $a"
  gh attestation verify "$TMP/$a" --repo "$REPO"

  want=$(awk -v f="$a" '$2 == f { print $1; exit }' "$TMP/checksums.txt")
  if ! printf '%s' "$want" | grep -Eq '^[0-9a-f]{64}$'; then
    echo "error: no valid checksum line for $a in checksums.txt" >&2
    exit 1
  fi
  got=$(sha256_of "$TMP/$a")
  if [ "$got" != "$want" ]; then
    echo "error: checksum mismatch for $a (expected $want, got $got)" >&2
    exit 1
  fi
  printf '%s  %s\n' "$want" "$a" >> "$TMP/pinned.txt"
done

mkdir -p "$RELEASE_DIR"
printf '%s\n' "$V" > "$TMP/version"
mv -f "$TMP/pinned.txt" "$RELEASE_DIR/checksums.txt"
mv -f "$TMP/version" "$RELEASE_DIR/version"

echo
echo "Pinned v$V: all 4 archives passed attestation and checksum verification."
echo "Commit release/version and release/checksums.txt in a pull request."
echo "Until it merges, the plugin keeps installing the previous pin."

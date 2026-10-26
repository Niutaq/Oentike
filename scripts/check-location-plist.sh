#!/bin/sh
# Contract check: macOS location capability requires usage strings and a stable bundle id.
# Safe to run on any OS — validates the source Info.plist, not a built binary.
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
plist="$root/oentike-web/src-tauri/Info.plist"
conf="$root/oentike-web/src-tauri/tauri.conf.json"

if [ ! -f "$plist" ]; then
  printf 'missing %s\n' "$plist" >&2
  exit 1
fi

require_key() {
  key=$1
  if ! grep -q "<key>$key</key>" "$plist"; then
    printf 'Info.plist missing required key: %s\n' "$key" >&2
    exit 1
  fi
}

require_key NSLocationWhenInUseUsageDescription
require_key NSLocationUsageDescription
require_key CFBundleIdentifier

if ! grep -q '<string>com.oentike.desktop</string>' "$plist"; then
  printf 'Info.plist CFBundleIdentifier must be com.oentike.desktop\n' >&2
  exit 1
fi

if [ -f "$conf" ] && ! grep -q '"identifier": "com.oentike.desktop"' "$conf"; then
  printf 'tauri.conf.json identifier must match Info.plist (com.oentike.desktop)\n' >&2
  exit 1
fi

printf 'location Info.plist contract ok\n'

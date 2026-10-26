#!/bin/sh
# Start Astro (hot reload) and a trusted Oentike.app shell for Core Location.
# The Tauri config must set the window URL to http://127.0.0.1:4321/... so the
# WebView loads the live server while the native shell stays a real .app bundle.
# Usage: scripts/run-macos-desktop.sh [tauri-config]
set -eu

if [ "$(uname -s)" != "Darwin" ]; then
  printf 'run-macos-desktop.sh is for macOS only\n' >&2
  exit 1
fi

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root/oentike-web"

config=${1:-src-tauri/tauri.field.conf.json}
base="http://127.0.0.1:4321"
app="src-tauri/target/debug/bundle/macos/Oentike.app"
binary="$app/Contents/MacOS/Oentike"
astro_log="${TMPDIR:-/tmp}/oentike-astro.log"

astro_ready() {
  curl --fail --silent --max-time 1 "$base/" >/dev/null 2>&1 \
    || curl --fail --silent --max-time 1 "http://localhost:4321/" >/dev/null 2>&1
}

if [ ! -x node_modules/.bin/tauri ]; then
  printf 'Run ./oentike setup first.\n' >&2
  exit 1
fi

astro_pid=""
stop_astro() {
  if [ -n "$astro_pid" ] && kill -0 "$astro_pid" 2>/dev/null; then
    kill "$astro_pid" 2>/dev/null || true
    wait "$astro_pid" 2>/dev/null || true
  fi
}
trap stop_astro EXIT INT TERM

if astro_ready; then
  printf 'Reusing Astro already on %s\n' "$base"
else
  printf 'Starting Astro on %s …\n' "$base"
  env -u npm_config_devdir npm run dev >"$astro_log" 2>&1 &
  astro_pid=$!
  deadline=$(( $(date +%s) + 60 ))
  until astro_ready; do
    if ! kill -0 "$astro_pid" 2>/dev/null; then
      printf 'Astro failed to start. Log (%s):\n' "$astro_log" >&2
      tail -n 40 "$astro_log" >&2 || true
      exit 1
    fi
    if [ "$(date +%s)" -ge "$deadline" ]; then
      printf 'Astro did not become ready on %s within 60s. Log (%s):\n' "$base" "$astro_log" >&2
      tail -n 40 "$astro_log" >&2 || true
      printf 'Hint: if another Astro is bound only to localhost/::1, stop it and retry.\n' >&2
      exit 1
    fi
    sleep 0.25
  done
  printf 'Astro ready on %s\n' "$base"
fi

need_build=1
if [ "${FORCE_BUILD:-}" != "1" ] && [ -x "$binary" ]; then
  need_build=0
  for source in \
    src-tauri/build.rs \
    src-tauri/src/location.rs \
    src-tauri/src/lib.rs \
    src-tauri/Cargo.toml \
    src-tauri/Cargo.lock \
    src-tauri/Info.plist \
    src-tauri/tauri.conf.json \
    src-tauri/capabilities/default.json \
    "$config"
  do
    if [ -e "$source" ] && [ "$source" -nt "$binary" ]; then
      need_build=1
      break
    fi
  done
fi

if [ "$need_build" = "1" ]; then
  printf 'Building trusted Oentike.app (first run or native sources changed; may take a few minutes) …\n'
  env -u npm_config_devdir npx tauri build --debug --bundles app --config "$config"
else
  printf 'Reusing existing Oentike.app (FORCE_BUILD=1 to rebuild)\n'
fi

if [ ! -d "$app" ]; then
  printf 'Missing %s after build\n' "$app" >&2
  exit 1
fi

open "$app"
printf 'Opened %s → %s\n' "$app" "$base"
printf 'Hot reload: Astro. GPS: .app bundle. Ctrl+C stops the UI server started by this script.\n'

if [ -n "$astro_pid" ]; then
  wait "$astro_pid"
else
  printf 'Press Ctrl+C when finished.\n'
  while true; do sleep 3600; done
fi

#!/bin/sh
set -u
failed=0
check() {
 if command -v "$1" >/dev/null 2>&1; then
  printf 'OK    %s\n' "$1"
 else
  printf 'MISSING  %s — %s\n' "$1" "$2"
  failed=1
 fi
}
check task 'install go-task or run mise install'
check go 'run mise install'
check cargo 'run mise install'
check node 'run mise install'
check npm 'npm is bundled with Node.js'
check docker 'install and start Docker with Compose'
check curl 'required for API readiness checks'
if command -v node >/dev/null 2>&1; then
 if ! node -e 'const [a,b]=process.versions.node.split(".").map(Number); process.exit(a>22 || (a===22 && b>=12) ? 0 : 1)'; then
  printf 'MISSING  Node.js >=22.12\n'; failed=1
 fi
fi
if command -v docker >/dev/null 2>&1; then
 if docker compose version >/dev/null 2>&1; then printf 'OK    Docker Compose\n'; else printf 'MISSING  Docker Compose\n'; failed=1; fi
 if docker info >/dev/null 2>&1; then printf 'OK    Docker daemon\n'; else printf 'MISSING  a running Docker daemon or permission to access it\n'; failed=1; fi
fi
if [ -d oentike-web/node_modules ]; then printf 'OK    UI dependencies installed (refresh: ./oentike setup)\n'; else printf 'INFO  Install UI dependencies: ./oentike setup\n'; fi
if [ "$(uname -s)" = Darwin ]; then
 if xcode-select -p >/dev/null 2>&1; then printf 'OK    Xcode Command Line Tools\n'; else printf 'MISSING  Xcode Command Line Tools: xcode-select --install\n'; failed=1; fi
fi
printf '\nTools should match the versions in mise.toml. Tauri system libraries are platform-specific.\n'
exit "$failed"

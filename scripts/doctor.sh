#!/bin/sh
set -u
failed=0
check() {
 if command -v "$1" >/dev/null 2>&1; then
  printf 'OK    %s\n' "$1"
 else
  printf 'BRAK  %s — %s\n' "$1" "$2"
  failed=1
 fi
}
check task 'zainstaluj go-task lub uruchom mise install'
check go 'uruchom mise install'
check cargo 'uruchom mise install'
check node 'uruchom mise install'
check npm 'npm jest dostarczany z Node.js'
check docker 'zainstaluj i uruchom Docker z Compose'
check curl 'wymagany do sprawdzania gotowości API'
if command -v node >/dev/null 2>&1; then
 if ! node -e 'const [a,b]=process.versions.node.split(".").map(Number); process.exit(a>22 || (a===22 && b>=12) ? 0 : 1)'; then
  printf 'BRAK  Node.js >=22.12\n'; failed=1
 fi
fi
if command -v docker >/dev/null 2>&1; then
 if docker compose version >/dev/null 2>&1; then printf 'OK    Docker Compose\n'; else printf 'BRAK  Docker Compose\n'; failed=1; fi
 if docker info >/dev/null 2>&1; then printf 'OK    Docker daemon\n'; else printf 'BRAK  działającego Docker daemon lub uprawnień do niego\n'; failed=1; fi
fi
if [ -d oentike-web/node_modules ]; then printf 'OK    zależności UI obecne (odśwież: ./oentike setup)\n'; else printf 'INFO  zależności UI: ./oentike setup\n'; fi
if [ "$(uname -s)" = Darwin ]; then
 if xcode-select -p >/dev/null 2>&1; then printf 'OK    Xcode Command Line Tools\n'; else printf 'BRAK  Xcode Command Line Tools: xcode-select --install\n'; failed=1; fi
fi
printf '\nNarzędzia powinny odpowiadać wersjom w mise.toml. Biblioteki systemowe Tauri zależą od platformy.\n'
exit "$failed"

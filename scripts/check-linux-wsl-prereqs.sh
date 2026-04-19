#!/usr/bin/env bash

set -u

failures=0
recommended_webkit_pkg='libwebkit2gtk-4.0-dev'

check_command() {
  local name="$1"
  if command -v "$name" >/dev/null 2>&1; then
    printf 'OK   command: %s\n' "$name"
    return
  fi

  printf 'MISS command: %s\n' "$name"
  failures=$((failures + 1))
}

check_pkg() {
  local label="$1"
  local package_name="$2"
  if pkg-config --exists "$package_name"; then
    printf 'OK   pkg-config: %s (%s)\n' "$label" "$(pkg-config --modversion "$package_name")"
    return 0
  fi

  printf 'MISS pkg-config: %s\n' "$label"
  return 1
}

printf 'Linux/WSL preflight for Wails\n'

check_command go
check_command npm
check_command gcc
check_command pkg-config
check_command wails

gtk_ok=0
webkit_ok=0

if check_pkg 'GTK3 development files' 'gtk+-3.0'; then
  gtk_ok=1
else
  failures=$((failures + 1))
fi

if check_pkg 'WebKitGTK 4.0 development files' 'webkit2gtk-4.0'; then
  webkit_ok=1
elif check_pkg 'WebKitGTK 4.1 development files' 'webkit2gtk-4.1'; then
  webkit_ok=1
  recommended_webkit_pkg='libwebkit2gtk-4.1-dev'
  printf 'INFO build with: wails dev -tags webkit2_41 (or wails build -tags webkit2_41)\n'
else
  if command -v apt-cache >/dev/null 2>&1 && apt-cache show libwebkit2gtk-4.1-dev >/dev/null 2>&1; then
    recommended_webkit_pkg='libwebkit2gtk-4.1-dev'
    printf 'INFO apt package available: %s\n' "$recommended_webkit_pkg"
  fi
  failures=$((failures + 1))
fi

if [ -n "${WSL_DISTRO_NAME:-}" ]; then
  printf 'INFO WSL distro: %s\n' "$WSL_DISTRO_NAME"
  if [ -n "${DISPLAY:-}${WAYLAND_DISPLAY:-}" ]; then
    printf 'OK   GUI bridge: DISPLAY/WAYLAND detected\n'
  else
    printf 'WARN GUI bridge: no DISPLAY or WAYLAND_DISPLAY detected\n'
  fi
fi

if [ "$gtk_ok" -eq 1 ] && [ "$webkit_ok" -eq 1 ]; then
  printf 'READY native Linux dependencies are present for Wails.\n'
else
  printf 'BLOCKED install the missing Linux development packages before running Wails dev/build.\n'
  printf 'MANUAL sudo apt update\n'
  printf 'MANUAL sudo apt install build-essential pkg-config libgtk-3-dev %s\n' "$recommended_webkit_pkg"
  if [ "$recommended_webkit_pkg" = 'libwebkit2gtk-4.1-dev' ]; then
    printf 'MANUAL run Wails with: ./scripts/wails-wsl.sh dev (or wails dev -tags webkit2_41)\n'
  fi
fi

exit "$failures"

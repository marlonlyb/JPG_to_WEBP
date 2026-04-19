#!/usr/bin/env bash

set -eu

resolve_wails() {
  if command -v wails >/dev/null 2>&1; then
    command -v wails
    return 0
  fi

  local user_wails="$HOME/go/bin/wails"
  if [ -x "$user_wails" ]; then
    printf '%s\n' "$user_wails"
    return 0
  fi

  printf 'Missing Wails CLI. Install it with:\n' >&2
  printf '  go install github.com/wailsapp/wails/v2/cmd/wails@latest\n' >&2
  exit 1
}

needs_webkit_41_tag() {
  pkg-config --exists webkit2gtk-4.1 && ! pkg-config --exists webkit2gtk-4.0
}

has_tags_arg() {
  for arg in "$@"; do
    case "$arg" in
      -tags|--tags|-tags=*|--tags=*)
        return 0
        ;;
    esac
  done

  return 1
}

if [ "$#" -eq 0 ]; then
  printf 'Usage: %s <wails-subcommand> [args...]\n' "$0" >&2
  exit 1
fi

wails_bin="$(resolve_wails)"
subcommand="$1"
shift

if { [ "$subcommand" = "dev" ] || [ "$subcommand" = "build" ]; } && needs_webkit_41_tag && ! has_tags_arg "$@"; then
  exec "$wails_bin" "$subcommand" -tags webkit2_41 "$@"
fi

exec "$wails_bin" "$subcommand" "$@"
